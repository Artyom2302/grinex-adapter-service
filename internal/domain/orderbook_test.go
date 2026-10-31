package domain

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func levels(prices ...int64) []Level {
	out := make([]Level, 0, len(prices))
	for _, p := range prices {
		out = append(out, Level{Price: decimal.NewFromInt(p)})
	}
	return out
}

func TestAvgNM(t *testing.T) {
	book := levels(10, 20, 30, 40)

	tests := []struct {
		name    string
		n, m    int
		want    decimal.Decimal
		wantErr error
	}{
		{name: "whole range 2..4", n: 2, m: 4, want: decimal.NewFromInt(30)},
		{name: "single position 3..3", n: 3, m: 3, want: decimal.NewFromInt(30)},
		{name: "n = 0", n: 0, m: 0, want: decimal.Zero, wantErr: ErrInvalidRange},
		{name: "m < n", n: 4, m: 2, want: decimal.NewFromInt(0), wantErr: ErrInvalidRange},
		{name: "m > len", n: 4, m: 10, want: decimal.NewFromInt(0), wantErr: ErrOutOfRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AvgNM(book, tt.n, tt.m)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got.Compare(tt.want) != 0 {
				t.Errorf("AvgNM() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTopN(t *testing.T) {
	book := levels(10, 20, 30, 40)

	tests := []struct {
		name    string
		n       int
		want    decimal.Decimal
		wantErr error
	}{
		{name: "top 2", n: 2, want: decimal.NewFromInt(20)},
		{name: "n = 0", n: 0, want: decimal.Zero, wantErr: ErrInvalidRange},
		{name: "n > len", n: 5, want: decimal.NewFromInt(0), wantErr: ErrOutOfRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TopN(book, tt.n)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got.Compare(tt.want) != 0 {
				t.Errorf("AvgNM() = %v, want %v", got, tt.want)
			}
		})
	}
}
