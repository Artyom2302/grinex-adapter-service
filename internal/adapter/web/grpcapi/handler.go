package grpcapi

import (
	"context"
	ratesv1 "grinex-adapter-service/gen/rates/v1"
)

var _ ratesv1.RatesServiceServer = (*GrpcHandler)(nil)

type GrpcHandler struct {
	ratesv1.UnimplementedRatesServiceServer
}

func NewGrpcHandler() *GrpcHandler {
	return &GrpcHandler{}
}

func (h *GrpcHandler) GetRates(context.Context, *ratesv1.GetRatesRequest) (*ratesv1.GetRatesResponse, error) {
	return &ratesv1.GetRatesResponse{}, nil
}
