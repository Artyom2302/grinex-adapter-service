package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"grinex-adapter-service/internal/application"
	"grinex-adapter-service/internal/domain"
)

var _ application.RateRepository = (*RatesRepository)(nil)

type RatesRepository struct {
	pool *pgxpool.Pool
}

func NewRatesRepository(pool *pgxpool.Pool) *RatesRepository {
	return &RatesRepository{pool: pool}
}

// Save inserts a calculated rate.
func (r *RatesRepository) Save(ctx context.Context, rate domain.Rate) error {
	const q = `
		INSERT INTO rates (ask, bid, method, n, m, requested_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	var m *int
	if rate.CalcMethod == domain.AVG_NM {
		m = &rate.M
	}

	if _, err := r.pool.Exec(ctx, q,
		rate.Ask, rate.Bid, int16(rate.CalcMethod), rate.N, m, rate.FetchedAt,
	); err != nil {
		return fmt.Errorf("insert rate: %w", err)
	}
	return nil
}
