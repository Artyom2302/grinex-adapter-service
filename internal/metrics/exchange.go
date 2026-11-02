package metrics

import (
	"context"
	"time"

	"grinex-adapter-service/internal/application"
	"grinex-adapter-service/internal/domain"
)

var _ application.ExchangeClient = (*InstrumentedExchange)(nil)

type InstrumentedExchange struct {
	next application.ExchangeClient
	m    *Metrics
}

func NewInstrumentedExchange(next application.ExchangeClient, m *Metrics) *InstrumentedExchange {
	return &InstrumentedExchange{next: next, m: m}
}

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
