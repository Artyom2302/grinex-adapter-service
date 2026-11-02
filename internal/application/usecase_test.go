package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"grinex-adapter-service/internal/domain"
)

type fakeExchange struct {
	book  domain.OrderBook
	err   error
	calls int
}

func (f *fakeExchange) GetOrderBook(_ context.Context) (domain.OrderBook, error) {
	f.calls++
	return f.book, f.err
}

type fakeRepo struct {
	saved []domain.Rate
	err   error
}

func (f *fakeRepo) Save(_ context.Context, rate domain.Rate) error {
	f.saved = append(f.saved, rate)
	return f.err
}

func lv(prices ...int64) []domain.Level {
	out := make([]domain.Level, 0, len(prices))
	for _, p := range prices {
		out = append(out, domain.Level{Price: decimal.NewFromInt(p)})
	}
	return out
}

func TestRatesService_GetRates(t *testing.T) {
	fixedNow := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	book := domain.OrderBook{
		Asks: lv(101, 102, 103),
		Bids: lv(99, 98, 97),
	}
	errExchange := errors.New("exchange down")
	errDB := errors.New("db down")

	tests := []struct {
		name          string
		query         GetRatesQuery
		exchangeErr   error
		repoErr       error
		wantErr       error
		wantAsk       decimal.Decimal
		wantBid       decimal.Decimal
		wantExchCalls int
		wantSaved     int
	}{
		{
			name:          "unknown method skips exchange",
			query:         GetRatesQuery{Method: domain.MethodUnspecified},
			wantErr:       domain.ErrUnknownMethod,
			wantExchCalls: 0,
			wantSaved:     0,
		},
		{
			name:          "top_n returns first level",
			query:         GetRatesQuery{Method: domain.MethodTopN, N: 1},
			wantAsk:       decimal.NewFromInt(101),
			wantBid:       decimal.NewFromInt(99),
			wantExchCalls: 1,
			wantSaved:     1,
		},
		{
			name:          "avg_nm averages range",
			query:         GetRatesQuery{Method: domain.MethodAvgNM, N: 1, M: 3},
			wantAsk:       decimal.NewFromInt(102),
			wantBid:       decimal.NewFromInt(98),
			wantExchCalls: 1,
			wantSaved:     1,
		},
		{
			name:          "exchange error is wrapped, nothing saved",
			query:         GetRatesQuery{Method: domain.MethodTopN, N: 1},
			exchangeErr:   errExchange,
			wantErr:       errExchange,
			wantExchCalls: 1,
			wantSaved:     0,
		},
		{
			name:          "save error is returned",
			query:         GetRatesQuery{Method: domain.MethodTopN, N: 1},
			repoErr:       errDB,
			wantErr:       errDB,
			wantExchCalls: 1,
			wantSaved:     1,
		},
		{
			name:          "position out of range, nothing saved",
			query:         GetRatesQuery{Method: domain.MethodTopN, N: 10},
			wantErr:       domain.ErrOutOfRange,
			wantExchCalls: 1,
			wantSaved:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ex := &fakeExchange{book: book, err: tt.exchangeErr}
			repo := &fakeRepo{err: tt.repoErr}
			svc := NewRatesService(ex, repo, func() time.Time { return fixedNow })

			got, err := svc.GetRates(context.Background(), tt.query)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if ex.calls != tt.wantExchCalls {
				t.Errorf("exchange calls = %d, want %d", ex.calls, tt.wantExchCalls)
			}
			if len(repo.saved) != tt.wantSaved {
				t.Errorf("saved = %d, want %d", len(repo.saved), tt.wantSaved)
			}
			if tt.wantErr != nil {
				return
			}

			if got.Ask.Compare(tt.wantAsk) != 0 {
				t.Errorf("ask = %v, want %v", got.Ask, tt.wantAsk)
			}
			if got.Bid.Compare(tt.wantBid) != 0 {
				t.Errorf("bid = %v, want %v", got.Bid, tt.wantBid)
			}
			if !got.FetchedAt.Equal(fixedNow) {
				t.Errorf("requested_at = %v, want %v", got.FetchedAt, fixedNow)
			}

			saved := repo.saved[0]
			if saved.Ask.Compare(got.Ask) != 0 || saved.Bid.Compare(got.Bid) != 0 || !saved.FetchedAt.Equal(got.FetchedAt) {
				t.Errorf("saved rate %+v differs from returned %+v", saved, got)
			}
		})
	}
}
