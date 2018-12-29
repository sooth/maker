package util

import "math"

func Roundx(val float64, x float64) float64 {
	return math.Round(val*x) / x
}

func Round8(val float64) float64 {
	return Roundx(val, 1/0.00000001)
}
