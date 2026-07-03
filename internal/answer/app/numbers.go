package app

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	citationPattern = regexp.MustCompile(`\[[A-Za-z]\d+\]`)
	numberPattern   = regexp.MustCompile(`[0-9]+(?:[.,][0-9]+)?`)
)

func numericLiterals(text string) []float64 {
	matches := numberPattern.FindAllString(text, -1)
	values := make([]float64, 0, len(matches))
	for _, match := range matches {
		normalized := strings.ReplaceAll(match, ",", ".")
		value, err := strconv.ParseFloat(normalized, 64)
		if err != nil {
			continue
		}
		values = append(values, value)
	}
	return values
}

func stripCitations(text string) string {
	return citationPattern.ReplaceAllString(text, "")
}

func isYear(value float64) bool {
	if value != float64(int64(value)) {
		return false
	}
	return value >= 1900 && value <= 2100
}

func approxEqual(a float64, b float64) bool {
	tolerance := 0.01 * absFloat(b)
	if tolerance < 1e-9 {
		tolerance = 1e-9
	}
	return absFloat(a-b) <= tolerance
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
