package app

import (
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type guardResult struct {
	numbersChecked int
	violations     int
}

func runGuard(summary string, pack *kmapv1.EvidencePack) guardResult {
	allowed := allowedNumbers(pack)
	result := guardResult{}
	for _, value := range numericLiterals(stripCitations(summary)) {
		if isYear(value) {
			continue
		}
		result.numbersChecked++
		if !containsApprox(allowed, value) {
			result.violations++
		}
	}
	return result
}

func allowedNumbers(pack *kmapv1.EvidencePack) []float64 {
	values := []float64{}
	collect := func(structValue *structpb.Struct) {
		values = append(values, numbersFromStruct(structValue)...)
	}
	for _, item := range pack.GetFacts() {
		collect(item.GetPayload())
	}
	for _, item := range pack.GetConsensus() {
		collect(item)
	}
	for _, item := range pack.GetContradictions() {
		collect(item)
	}
	for _, item := range pack.GetGaps() {
		collect(item)
	}
	for _, item := range pack.GetExperts() {
		collect(item.GetEvidence())
	}
	return values
}

func numbersFromStruct(structValue *structpb.Struct) []float64 {
	if structValue == nil {
		return nil
	}
	var values []float64
	for _, value := range structValue.GetFields() {
		values = append(values, numbersFromValue(value)...)
	}
	return values
}

func numbersFromValue(value *structpb.Value) []float64 {
	switch kind := value.GetKind().(type) {
	case *structpb.Value_NumberValue:
		return []float64{kind.NumberValue}
	case *structpb.Value_StringValue:
		return numericLiterals(kind.StringValue)
	case *structpb.Value_StructValue:
		return numbersFromStruct(kind.StructValue)
	case *structpb.Value_ListValue:
		var values []float64
		for _, item := range kind.ListValue.GetValues() {
			values = append(values, numbersFromValue(item)...)
		}
		return values
	default:
		return nil
	}
}

func containsApprox(values []float64, target float64) bool {
	for _, value := range values {
		if approxEqual(target, value) {
			return true
		}
	}
	return false
}
