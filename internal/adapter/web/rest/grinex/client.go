package grinex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"grinex-adapter-service/internal/application"
	"grinex-adapter-service/internal/domain"
)

// Errors returned by the Grinex client.
var (
	ErrUnexpectedStatus  = errors.New("unexpected response status")
	ErrUnexpectedContent = errors.New("unexpected response content type")
)

const depthPath = "/api/v2/depth"

var _ application.ExchangeClient = (*Client)(nil)

type Client struct {
	http   *resty.Client
	market string
}

func NewClient(baseURL, market string, timeout time.Duration) *Client {
	return &Client{
		http:   resty.New().SetBaseURL(baseURL).SetTimeout(timeout).SetTransport(otelhttp.NewTransport(http.DefaultTransport)),
		market: market,
	}
}

type depthResponse struct {
	Timestamp int64      `json:"timestamp"`
	Asks      []levelDTO `json:"asks"`
	Bids      []levelDTO `json:"bids"`
}

type levelDTO struct {
	Price  string `json:"price"`
	Volume string `json:"volume"`
}

func (c *Client) GetOrderBook(ctx context.Context) (domain.OrderBook, error) {
	var body depthResponse

	resp, err := c.http.R().
		SetContext(ctx).
		SetQueryParam("market", c.market).
		SetResult(&body).
		Get(depthPath)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf("request depth: %w", err)
	}
	if resp.IsError() {
		return domain.OrderBook{}, fmt.Errorf("%w: %d", ErrUnexpectedStatus, resp.StatusCode())
	}
	if ct := resp.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		return domain.OrderBook{}, fmt.Errorf("%w: %q", ErrUnexpectedContent, ct)
	}

	asks, err := toLevels(body.Asks)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf("parse asks: %w", err)
	}
	bids, err := toLevels(body.Bids)
	if err != nil {
		return domain.OrderBook{}, fmt.Errorf("parse bids: %w", err)
	}

	slices.SortFunc(asks, func(a, b domain.Level) int { return a.Price.Compare(b.Price) })
	slices.SortFunc(bids, func(a, b domain.Level) int { return b.Price.Compare(a.Price) })

	return domain.OrderBook{Asks: asks, Bids: bids}, nil
}

func toLevels(dtos []levelDTO) ([]domain.Level, error) {
	levels := make([]domain.Level, 0, len(dtos))
	for i, d := range dtos {
		price, err := decimal.NewFromString(d.Price)
		if err != nil {
			return nil, fmt.Errorf("level %d price %q: %w", i, d.Price, err)
		}
		volume, err := decimal.NewFromString(d.Volume)
		if err != nil {
			return nil, fmt.Errorf("level %d volume %q: %w", i, d.Volume, err)
		}
		levels = append(levels, domain.Level{Price: price, Volume: volume})
	}
	return levels, nil
}
