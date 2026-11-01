package main

import (
	"context"
	"grinex-adapter-service/internal/adapter/storage/postgres"
	"grinex-adapter-service/internal/adapter/web/rest/grinex"
	"grinex-adapter-service/internal/config"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	ratesv1 "grinex-adapter-service/gen/rates/v1"
	"grinex-adapter-service/internal/adapter/web/grpcapi"
	"grinex-adapter-service/internal/application"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		log.Fatal("config error: ", err)
	}

	pool, err := postgres.NewPool(ctx, cfg.DSN)
	if err != nil {
		log.Fatal("failed to connect to postgres:", err)
	}
	defer pool.Close()

	applied, err := postgres.Migrate(ctx, pool)
	if err != nil {
		log.Fatalf("migrate: %v", err)
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
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("grpc server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("grpc server stopped: %v", err)
	}
}
