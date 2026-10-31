package domain

import (
	"errors"

	"github.com/shopspring/decimal"
)

// Sentinel errors returned by rate calculations.
var (
	ErrEmptyLevels  = errors.New("order book side is empty")
	ErrOutOfRange   = errors.New("position is out of range")
	ErrInvalidRange = errors.New("invalid range: n must be <= m")
)

// Level is a single price level of an order book side.
type Level struct {
	Price  decimal.Decimal
	Volume decimal.Decimal
}

// OrderBook holds both sides of the spot order book.
type OrderBook struct {
	Asks []Level
	Bids []Level
}

// TopN returns the price at 1-based position n.
func TopN(levels []Level, n int) (decimal.Decimal, error) {
	if len(levels) == 0 {
		return decimal.Zero, ErrEmptyLevels
	}
	if n < 1 {
		return decimal.Zero, ErrInvalidRange
	}
	if n > len(levels) {
		return decimal.Zero, ErrOutOfRange
	}
	return levels[n-1].Price, nil
}

// AvgNM returns the average price over 1-based positions [n; m] inclusive.
func AvgNM(levels []Level, n, m int) (decimal.Decimal, error) {
	if len(levels) == 0 {
		return decimal.Zero, ErrEmptyLevels
	}
	if n < 1 {
		return decimal.Zero, ErrInvalidRange
	}
	if n > len(levels) || m > len(levels) {
		return decimal.Zero, ErrOutOfRange
	}
	if m < n {
		return decimal.Zero, ErrInvalidRange
	}
	var sum decimal.Decimal
	var count int64
	for i := n; i <= m; i++ {
		count++
		sum = sum.Add(levels[i-1].Price)
	}
	return sum.Div(decimal.NewFromInt(count)), nil
}
