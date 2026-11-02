package application

import (
	"context"
	"errors"
	"fmt"
	"grinex-adapter-service/internal/domain"
	"time"

	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("grinex-adapter-service/internal/application")
var ErrExchangeUnavailable = errors.New("exchange unavailable")

// ExchangeClient fetches the current order book from an exchange.
type ExchangeClient interface {
	GetOrderBook(ctx context.Context) (domain.OrderBook, error)
}

// RateRepository persists calculated rates.
type RateRepository interface {
	Save(ctx context.Context, rate domain.Rate) error
}

// GetRatesQuery is the input of the GetRates use case.
type GetRatesQuery struct {
	Method domain.CalcMethod
	N, M   int
}

// RatesService implements the GetRates use case.
type RatesService struct {
	exchange ExchangeClient
	repo     RateRepository
	now      func() time.Time
}

// NewRatesService creates a RatesService.
func NewRatesService(exchange ExchangeClient, repo RateRepository, now func() time.Time) *RatesService {
	return &RatesService{exchange: exchange, repo: repo, now: now}
}

// GetRates fetches the order book, calculates ask/bid with the requested method and stores the result.
// GetRates fetches the order book, calculates ask/bid with the requested method and stores the result.
func (s *RatesService) GetRates(ctx context.Context, q GetRatesQuery) (_ domain.Rate, err error) {
	ctx, span := tracer.Start(ctx, "RatesService.GetRates",
		trace.WithAttributes(
			attribute.Int("rates.method", int(q.Method)),
			attribute.Int("rates.n", q.N),
			attribute.Int("rates.m", q.M),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	if q.Method != domain.TOP_N && q.Method != domain.AVG_NM {
		return domain.Rate{}, fmt.Errorf("%w: %d", domain.ErrUnknownMethod, q.Method)
	}

	book, err := s.exchange.GetOrderBook(ctx)
	if err != nil {
		return domain.Rate{}, fmt.Errorf("get order book: %w", err)
	}

	var ask, bid decimal.Decimal
	switch q.Method {
	case domain.TOP_N:
		if ask, err = domain.TopN(book.Asks, q.N); err != nil {
			return domain.Rate{}, fmt.Errorf("convert asks: %w", err)
		}
		if bid, err = domain.TopN(book.Bids, q.N); err != nil {
			return domain.Rate{}, fmt.Errorf("convert bids: %w", err)
		}
	case domain.AVG_NM:
		if ask, err = domain.AvgNM(book.Asks, q.N, q.M); err != nil {
			return domain.Rate{}, fmt.Errorf("convert asks: %w", err)
		}
		if bid, err = domain.AvgNM(book.Bids, q.N, q.M); err != nil {
			return domain.Rate{}, fmt.Errorf("convert bids: %w", err)
		}
	default:
		return domain.Rate{}, fmt.Errorf("%w: %d", domain.ErrUnknownMethod, q.Method)
	}

	rate := domain.Rate{
		Ask:        ask,
		Bid:        bid,
		FetchedAt:  s.now(),
		CalcMethod: q.Method,
		N:          q.N,
		M:          q.M,
	}

	if err := s.repo.Save(ctx, rate); err != nil {
		return domain.Rate{}, fmt.Errorf("save rate: %w", err)
	}
	return rate, nil
}
