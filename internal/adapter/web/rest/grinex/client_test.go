package grinex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"grinex-adapter-service/internal/domain"
)

func newTestServer(t *testing.T, status int, contentType string, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != depthPath {
			t.Errorf("path = %q, want %q", r.URL.Path, depthPath)
		}
		if got := r.URL.Query().Get("market"); got != "usdta7a5" {
			t.Errorf("market = %q, want %q", got, "usdta7a5")
		}
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func assertPrices(t *testing.T, side string, levels []domain.Level, want ...string) {
	t.Helper()
	if len(levels) != len(want) {
		t.Fatalf("%s: len = %d, want %d", side, len(levels), len(want))
	}
	for i, w := range want {
		if !levels[i].Price.Equal(decimal.RequireFromString(w)) {
			t.Errorf("%s[%d] = %s, want %s", side, i, levels[i].Price, w)
		}
	}
}

func TestClient_GetOrderBook_OK(t *testing.T) {
	fixture, err := os.ReadFile("testdata/depth.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := newTestServer(t, http.StatusOK, "application/json; charset=utf-8", fixture)
	c := NewClient(srv.URL, "usdta7a5", time.Second)

	book, err := c.GetOrderBook(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertPrices(t, "asks", book.Asks, "80.15", "80.20", "80.30")
	assertPrices(t, "bids", book.Bids, "80.05", "80.00", "79.90")
}

func TestClient_GetOrderBook_Errors(t *testing.T) {
	validJSON := `{"timestamp":1,"asks":[{"price":"1","volume":"1"}],"bids":[{"price":"1","volume":"1"}]}`

	tests := []struct {
		name        string
		status      int
		contentType string
		body        string
		wantErr     error // nil means "any error"
	}{
		{name: "403 html from WAF", status: http.StatusForbidden, contentType: "text/html", body: "<h1>Forbidden</h1>", wantErr: ErrUnexpectedStatus},
		{name: "500 json", status: http.StatusInternalServerError, contentType: "application/json", body: `{"error":"x"}`, wantErr: ErrUnexpectedStatus},
		{name: "200 html", status: http.StatusOK, contentType: "text/html", body: "<html></html>", wantErr: ErrUnexpectedContent},
		{name: "200 json without content type", status: http.StatusOK, contentType: "", body: validJSON, wantErr: ErrUnexpectedContent},
		{name: "invalid price", status: http.StatusOK, contentType: "application/json", body: `{"asks":[{"price":"abc","volume":"1"}],"bids":[]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(t, tt.status, tt.contentType, []byte(tt.body))
			c := NewClient(srv.URL, "usdta7a5", time.Second)

			_, err := c.GetOrderBook(context.Background())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_GetOrderBook_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)
	c := NewClient(srv.URL, "usdta7a5", 50*time.Millisecond)

	if _, err := c.GetOrderBook(context.Background()); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
