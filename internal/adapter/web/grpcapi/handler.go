// Package grpcapi exposes the rates use case over gRPC.
package grpcapi

import (
	"context"
	"errors"
	ratesv1 "grinex-adapter-service/gen/rates/v1"
	"grinex-adapter-service/internal/application"
	"grinex-adapter-service/internal/domain"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ ratesv1.RatesServiceServer = (*GrpcHandler)(nil)

// RatesGetter is the use case the gRPC handler depends on.
type RatesGetter interface {
	GetRates(ctx context.Context, q application.GetRatesQuery) (domain.Rate, error)
}

// GrpcHandler implements ratesv1.RatesServiceServer.
type GrpcHandler struct {
	ratesv1.UnimplementedRatesServiceServer
	rates  RatesGetter
	logger *zap.Logger
}

// NewGrpcHandler creates a GrpcHandler.
func NewGrpcHandler(rates RatesGetter, logger *zap.Logger) *GrpcHandler {
	return &GrpcHandler{rates: rates, logger: logger}
}

// GetRates calculates ask/bid for the requested method and maps errors to gRPC codes.
func (h *GrpcHandler) GetRates(ctx context.Context, req *ratesv1.GetRatesRequest) (*ratesv1.GetRatesResponse, error) {
	rate, err := h.rates.GetRates(ctx, application.GetRatesQuery{
		Method: toDomainMethod(req.GetCalcMethod()),
		N:      int(req.GetN()),
		M:      int(req.GetM()),
	})
	if err != nil {
		return nil, h.toStatus(err)
	}

	return &ratesv1.GetRatesResponse{
		Ask:       rate.Ask.InexactFloat64(),
		Bid:       rate.Bid.InexactFloat64(),
		FetchedAt: timestamppb.New(rate.FetchedAt),
	}, nil
}

// toDomainMethod maps the proto enum to the domain calculation method.
func toDomainMethod(m ratesv1.CALCMETHOD) domain.CalcMethod {
	switch m {
	case ratesv1.CALCMETHOD_CALCMETHOD_AVG_NM:
		return domain.MethodAvgNM
	case ratesv1.CALCMETHOD_CALCMETHOD_TOP_N:
		return domain.MethodTopN
	default:
		return domain.MethodUnspecified
	}
}

func (h *GrpcHandler) toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrUnknownMethod) || errors.Is(err, domain.ErrOutOfRange) || errors.Is(err, domain.ErrInvalidRange) || errors.Is(err, domain.ErrInvalidRange):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrEmptyLevels):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, application.ErrExchangeUnavailable):
		h.logger.Warn("exchange unavailable", zap.Error(err))
		return status.Error(codes.Unavailable, "exchange unavailable")
	default:
		h.logger.Error("get rates failed", zap.Error(err))
		return status.Error(codes.Internal, "internal error")
	}
}
