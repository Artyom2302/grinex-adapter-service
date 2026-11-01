package main

import (
	"context"
	"fmt"
	ratesv1 "grinex-adapter-service/gen/rates/v1"
	"grinex-adapter-service/internal/adapter/storage/postgres"
	"grinex-adapter-service/internal/adapter/web/grpcapi"
	"grinex-adapter-service/internal/adapter/web/rest/grinex"
	"grinex-adapter-service/internal/application"
	"grinex-adapter-service/internal/config"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	pool, err := postgres.NewPool(ctx, cfg.DSN)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer pool.Close()

	applied, err := postgres.Migrate(ctx, pool)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Printf("migrations applied: %v", applied)

	repo := postgres.NewRatesRepository(pool)
	var exchange application.ExchangeClient
	if cfg.Grinex.UseStub {
		log.Printf("using stub exchange")
		exchange = stubExchange{}
	} else {
		exchange = grinex.NewClient(cfg.Grinex.BaseURL, cfg.Grinex.Market, cfg.Grinex.Timeout)
	}

	svc := application.NewRatesService(exchange, repo, time.Now)

	grpcServer := grpc.NewServer()
	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthcheck)
	ratesv1.RegisterRatesServiceServer(grpcServer, grpcapi.NewGrpcHandler(svc))
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("grpc server listening at %v", lis.Addr())
		serveErr <- grpcServer.Serve(lis)
	}()
	select {
	case err := <-serveErr:
		log.Printf("serve error: %v", err)
		return fmt.Errorf("serve error: %w", err)
	case <-ctx.Done():
		log.Printf("shutdown signal received")
	}
	healthcheck.Shutdown()
	timer := time.AfterFunc(cfg.AppTimeout, func() {
		grpcServer.Stop()
	})
	defer timer.Stop()
	grpcServer.GracefulStop()
	log.Printf("server stopped")
	return nil
}
