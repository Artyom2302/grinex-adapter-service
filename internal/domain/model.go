package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var ErrUnknownMethod = errors.New("unknown calculation method")

const (
	NO_SPECIFIED CalcMethod = iota
	TOP_N
	AVG_NM
)

type CalcMethod int

type Rate struct {
	Ask        decimal.Decimal
	Bid        decimal.Decimal
	FetchedAt  time.Time
	CalcMethod CalcMethod
	N, M       int
}
