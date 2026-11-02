// Command app runs the gRPC rates service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	ratesv1 "grinex-adapter-service/gen/rates/v1"
	"grinex-adapter-service/internal/adapter/storage/postgres"
	"grinex-adapter-service/internal/adapter/web/grpcapi"
	"grinex-adapter-service/internal/adapter/web/rest/grinex"
	"grinex-adapter-service/internal/application"
	"grinex-adapter-service/internal/config"
	"grinex-adapter-service/internal/metrics"
	"grinex-adapter-service/internal/telemetry"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	shutdownTracer, err := telemetry.InitTracer(ctx, "grinex-adapter-service", cfg.OTLPEndpoint)
	if err != nil {
		return fmt.Errorf("init tracer: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracer(shutdownCtx); err != nil {
			logger.Warn("tracer shutdown failed", zap.Error(err))
		}
	}()

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	m := metrics.New(reg)

	pool, err := postgres.NewPool(ctx, cfg.DSN)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	applied, err := postgres.Migrate(ctx, pool)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	logger.Info("migrations applied", zap.Strings("files", applied))

	var exchange application.ExchangeClient
	if cfg.Grinex.UseStub {
		logger.Warn("using stub exchange")
		exchange = stubExchange{}
	} else {
		exchange = grinex.NewClient(cfg.Grinex.BaseURL, cfg.Grinex.Market, cfg.Grinex.Timeout)
	}
	exchange = metrics.NewInstrumentedExchange(exchange, m)

	repo := postgres.NewRatesRepository(pool)
	svc := application.NewRatesService(exchange, repo, time.Now)

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			grpcapi.MetricsInterceptor(m),
			grpcapi.LoggingInterceptor(logger),
		),
	)
	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthcheck)
	ratesv1.RegisterRatesServiceServer(grpcServer, grpcapi.NewGrpcHandler(svc, logger))
	reflection.Register(grpcServer)

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.GRPCAddr, err)
	}

	metricsSrv := &http.Server{
		Addr:              cfg.MetricsAddr,
		Handler:           promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 2)
	go func() {
		logger.Info("grpc server listening", zap.String("addr", lis.Addr().String()))
		serveErr <- grpcServer.Serve(lis)
	}()
	go func() {
		logger.Info("metrics server listening", zap.String("addr", cfg.MetricsAddr))
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("metrics server: %w", err)
		}
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	healthcheck.Shutdown()
	timer := time.AfterFunc(cfg.AppTimeout, func() {
		logger.Warn("graceful stop timed out, forcing stop")
		grpcServer.Stop()
	})
	defer timer.Stop()
	grpcServer.GracefulStop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("metrics server shutdown failed", zap.Error(err))
	}

	logger.Info("server stopped")
	return nil
}
