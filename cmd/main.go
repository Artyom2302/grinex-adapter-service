package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	repo := postgres.NewRatesRepository(pool)
	svc := application.NewRatesService(exchange, repo, time.Now)

	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(grpcapi.LoggingInterceptor(logger)))
	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthcheck)
	ratesv1.RegisterRatesServiceServer(grpcServer, grpcapi.NewGrpcHandler(svc, logger))
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.GRPCAddr, err)
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("grpc server listening", zap.String("addr", lis.Addr().String()))
		serveErr <- grpcServer.Serve(lis)
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

	logger.Info("server stopped")
	return nil
}
