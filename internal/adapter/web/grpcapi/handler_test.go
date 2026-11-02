package grpcapi

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ratesv1 "grinex-adapter-service/gen/rates/v1"
	"grinex-adapter-service/internal/application"
	"grinex-adapter-service/internal/domain"
)

type fakeRates struct {
	rate domain.Rate
	err  error
	got  application.GetRatesQuery
}

func (f *fakeRates) GetRates(ctx context.Context, q application.GetRatesQuery) (domain.Rate, error) {
	f.got = q
	return f.rate, f.err
}

func TestGrpcHandler_GetRates_ErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{name: "unknown method", err: domain.ErrUnknownMethod, wantCode: codes.InvalidArgument},
		{name: "out of range", err: fmt.Errorf("convert asks: %w", domain.ErrOutOfRange), wantCode: codes.InvalidArgument},
		{name: "invalid range", err: domain.ErrInvalidRange, wantCode: codes.InvalidArgument},
		{name: "empty book", err: domain.ErrEmptyLevels, wantCode: codes.Unavailable},
		{name: "exchange down", err: fmt.Errorf("get order book: %w", application.ErrExchangeUnavailable), wantCode: codes.Unavailable},
		{name: "unexpected", err: errors.New("db down"), wantCode: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewGrpcHandler(&fakeRates{err: tt.err}, zap.NewNop())

			_, err := h.GetRates(context.Background(), &ratesv1.GetRatesRequest{})

			if got := status.Code(err); got != tt.wantCode {
				t.Errorf("code = %v, want %v (err: %v)", got, tt.wantCode, err)
			}
		})
	}
}

func TestGrpcHandler_GetRates_InternalHidesDetails(t *testing.T) {
	h := NewGrpcHandler(&fakeRates{err: errors.New("dsn=postgres://secret")}, zap.NewNop())

	_, err := h.GetRates(context.Background(), &ratesv1.GetRatesRequest{})

	if msg := status.Convert(err).Message(); msg != "internal error" {
		t.Errorf("message = %q, want %q", msg, "internal error")
	}
}

func TestGrpcHandler_GetRates_OK(t *testing.T) {
	at := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	fake := &fakeRates{rate: domain.Rate{
		Ask:       decimal.RequireFromString("80.15"),
		Bid:       decimal.RequireFromString("80.05"),
		FetchedAt: at,
	}}
	h := NewGrpcHandler(fake, zap.NewNop())

	resp, err := h.GetRates(context.Background(), &ratesv1.GetRatesRequest{
		CalcMethod: ratesv1.CALCMETHOD_CALCMETHOD_AVG_NM,
		N:          1,
		M:          3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}


	want := application.GetRatesQuery{Method: domain.AVG_NM, N: 1, M: 3}
	if fake.got != want {
		t.Errorf("query = %+v, want %+v", fake.got, want)
	}


	if resp.GetAsk() != 80.15 || resp.GetBid() != 80.05 {
		t.Errorf("ask/bid = %v/%v, want 80.15/80.05", resp.GetAsk(), resp.GetBid())
	}
	if !resp.GetFetchedAt().AsTime().Equal(at) {
		t.Errorf("fetched_at = %v, want %v", resp.GetFetchedAt().AsTime(), at)
	}
}

func TestToDomainMethod(t *testing.T) {
	tests := map[ratesv1.CALCMETHOD]domain.CalcMethod{
		ratesv1.CALCMETHOD_CALCMETHOD_UNSPECIFIED: domain.NO_SPECIFIED,
		ratesv1.CALCMETHOD_CALCMETHOD_TOP_N:       domain.TOP_N,
		ratesv1.CALCMETHOD_CALCMETHOD_AVG_NM:      domain.AVG_NM,
		ratesv1.CALCMETHOD(99):                    domain.NO_SPECIFIED,
	}
	for in, want := range tests {
		if got := toDomainMethod(in); got != want {
			t.Errorf("toDomainMethod(%v) = %v, want %v", in, got, want)
		}
	}
}
