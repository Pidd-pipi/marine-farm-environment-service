package service

import (
	"math"
	"strconv"
)

// f2 formats a float with two decimals for audit detail lines.
func f2(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// itoa formats an integer for audit detail lines.
func itoa(n int) string {
	return strconv.Itoa(n)
}

// finiteNonNegative reports whether v is a finite, non-negative number.
func finiteNonNegative(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0
}

// finite reports whether v is neither NaN nor infinity.
func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
