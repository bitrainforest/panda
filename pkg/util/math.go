package util

import "math"

func Pow2(exp int64) int64 {
	return int64(math.Pow(float64(2), float64(exp)))
}
