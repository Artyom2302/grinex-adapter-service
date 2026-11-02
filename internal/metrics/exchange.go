package metrics

import (
	"context"
	"time"

	"grinex-adapter-service/internal/application"
	"grinex-adapter-service/internal/domain"
)

var _ application.ExchangeClient = (*InstrumentedExchange)(nil)

// InstrumentedExchange records metrics around an ExchangeClient.
type InstrumentedExchange struct {
	next application.ExchangeClient
	m    *Metrics
}

// NewInstrumentedExchange wraps next with metrics.
func NewInstrumentedExchange(next application.ExchangeClient, m *Metrics) *InstrumentedExchange {
	return &InstrumentedExchange{next: next, m: m}
}

// GetOrderBook calls the wrapped client and records duration and result.
func (e *InstrumentedExchange) GetOrderBook(ctx context.Context) (domain.OrderBook, error) {
	start := time.Now()
	book, err := e.next.GetOrderBook(ctx)
	e.m.ExchangeDuration.Observe(time.Since(start).Seconds())

	result := "ok"
	if err != nil {
		result = "error"
	}
	e.m.ExchangeRequests.WithLabelValues(result).Inc()

	return book, err
}
