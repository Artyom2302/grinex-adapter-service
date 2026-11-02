package main

import (
	"context"

	"github.com/shopspring/decimal"

	"grinex-adapter-service/internal/domain"
)

// stubExchange returns a fixed order book until the Grinex adapter exists (layer 4).
type stubExchange struct{}

func (stubExchange) GetOrderBook(_ context.Context) (domain.OrderBook, error) {
	lvl := func(price string) domain.Level {
		return domain.Level{Price: decimal.RequireFromString(price), Volume: decimal.NewFromInt(1000)}
	}
	return domain.OrderBook{
		Asks: []domain.Level{lvl("80.15"), lvl("80.20"), lvl("80.30")},
		Bids: []domain.Level{lvl("80.05"), lvl("80.00"), lvl("79.90")},
	}, nil
}
