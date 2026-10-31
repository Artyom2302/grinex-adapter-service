package main

import (
	ratesv1 "grinex-adapter-service/gen/rates/v1"
	"grinex-adapter-service/internal/adapter/web/grpcapi"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {

	grpcServer := grpc.NewServer()
	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthcheck)
	ratesv1.RegisterRatesServiceServer(grpcServer, grpcapi.NewGrpcHandler())
	reflection.Register(grpcServer)
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()
	err = grpcServer.Serve(lis)
	if err != nil {
		return
	}
	log.Printf("grpc server listening at %v", lis.Addr())

}
