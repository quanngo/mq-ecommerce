package utils

import (
	"math"
	"strings"
)

func String(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func LikeString(input string) string {
	s := strings.TrimSpace(input)
	return "%" + s + "%"
}

func RoundNumber(num float64) float64 {
	output := math.Pow(10, float64(4)) // 4f
	return float64(math.Round(num*output)) / output
}
