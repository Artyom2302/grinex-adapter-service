package application

import (
	"context"
	"fmt"
	"grinex-adapter-service/internal/domain"
	"time"

	"github.com/shopspring/decimal"
)

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
func (s *RatesService) GetRates(ctx context.Context, q GetRatesQuery) (domain.Rate, error) {
	if q.Method != domain.TOP_N && q.Method != domain.AVG_NM {
		return domain.Rate{}, fmt.Errorf("%w: %d", domain.ErrUnknownMethod, q.Method)
	}

	book, err := s.exchange.GetOrderBook(ctx)
	if err != nil {
		return domain.Rate{}, fmt.Errorf("get order book: %w", err)
	}

	var ask decimal.Decimal
	var bid decimal.Decimal
	switch q.Method {
	case domain.TOP_N:
		ask, err = domain.TopN(book.Asks, q.N)
		if err != nil {
			return domain.Rate{}, fmt.Errorf("convert asks: %w", err)
		}
		bid, err = domain.TopN(book.Bids, q.N)
		if err != nil {
			return domain.Rate{}, fmt.Errorf("convert bids: %w", err)
		}
	case domain.AVG_NM:
		ask, err = domain.AvgNM(book.Asks, q.N, q.M)
		if err != nil {
			return domain.Rate{}, fmt.Errorf("convert asks: %w", err)
		}
		bid, err = domain.AvgNM(book.Bids, q.N, q.M)
		if err != nil {
			return domain.Rate{}, fmt.Errorf("convert bids: %w", err)
		}
	default:
		return domain.Rate{}, fmt.Errorf("unknown method")
	}

	rate := domain.Rate{
		Ask:       ask,
		Bid:       bid,
		FetchedAt: s.now(),
	}

	if err := s.repo.Save(ctx, rate); err != nil {
		return domain.Rate{}, fmt.Errorf("save rate: %w", err)
	}
	return rate, nil

}
