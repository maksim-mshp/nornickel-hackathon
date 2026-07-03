package app

import (
	"slices"
	"strconv"
)

type NumericBand struct {
	Edges []float64 `koanf:"edges"`
}

type Bands struct {
	Numeric     map[string]NumericBand `koanf:"numeric"`
	Categorical map[string][]string    `koanf:"categorical"`
}

const unspecifiedBand = "unspecified"

func (bands Bands) ClassifyConditions(conditions map[string]any) map[string]string {
	class := make(map[string]string)
	for key, band := range bands.Numeric {
		if value, ok := numericConditionValue(conditions[key]); ok {
			class[key] = numericBandLabel(band.Edges, value)
		} else {
			class[key] = unspecifiedBand
		}
	}
	for key, values := range bands.Categorical {
		class[key] = categoricalBandLabel(values, stringCondition(conditions[key]))
	}
	return class
}

func numericBandLabel(edges []float64, value float64) string {
	if len(edges) == 0 {
		return unspecifiedBand
	}
	index := 0
	for index < len(edges) && value >= edges[index] {
		index++
	}
	switch index {
	case 0:
		return "<" + formatEdge(edges[0])
	case len(edges):
		return ">" + formatEdge(edges[len(edges)-1])
	default:
		return formatEdge(edges[index-1]) + "-" + formatEdge(edges[index])
	}
}

func categoricalBandLabel(values []string, value string) string {
	if value != "" && slices.Contains(values, value) {
		return value
	}
	return unspecifiedBand
}

func numericConditionValue(raw any) (float64, bool) {
	switch value := raw.(type) {
	case float64:
		return value, true
	case int:
		return float64(value), true
	case string:
		parsed, err := strconv.ParseFloat(value, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func stringCondition(raw any) string {
	value, _ := raw.(string)
	return value
}

func formatEdge(edge float64) string {
	return strconv.FormatFloat(edge, 'g', -1, 64)
}
