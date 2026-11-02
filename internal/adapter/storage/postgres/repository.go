package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"grinex-adapter-service/internal/application"
	"grinex-adapter-service/internal/domain"
)

var _ application.RateRepository = (*RatesRepository)(nil)
var tracer = otel.Tracer("grinex-adapter-service/internal/adapter/storage/postgres")

// RatesRepository stores calculated rates in PostgreSQL.
type RatesRepository struct {
	pool *pgxpool.Pool
}

// NewRatesRepository creates a RatesRepository backed by pool.
func NewRatesRepository(pool *pgxpool.Pool) *RatesRepository {
	return &RatesRepository{pool: pool}
}

// Save inserts a calculated rate.
func (r *RatesRepository) Save(ctx context.Context, rate domain.Rate) error {
	ctx, span := tracer.Start(ctx, "postgres.Save",
		trace.WithAttributes(attribute.String("db.system", "postgresql")),
	)
	defer span.End()
	const q = `
		INSERT INTO rates (ask, bid, method, n, m, requested_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	var m *int
	if rate.CalcMethod == domain.MethodAvgNM {
		m = &rate.M
	}

	if _, err := r.pool.Exec(ctx, q,
		rate.Ask, rate.Bid, rate.CalcMethod, rate.N, m, rate.FetchedAt,
	); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("insert rate: %w", err)
	}
	return nil
}
