// Package domain holds rate types and calculation rules.
package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// ErrUnknownMethod is returned for an unsupported calculation method.
var ErrUnknownMethod = errors.New("unknown calculation method")

// Calculation methods.
const (
	MethodUnspecified CalcMethod = iota
	MethodTopN
	MethodAvgNM
)

// CalcMethod selects how ask/bid are calculated from the order book.
type CalcMethod int16

// Rate is a calculated ask/bid pair.
type Rate struct {
	Ask        decimal.Decimal
	Bid        decimal.Decimal
	FetchedAt  time.Time
	CalcMethod CalcMethod
	N, M       int
}
