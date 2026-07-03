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
	var values []float64
	addRange := func(structValue *structpb.Struct) {
		if structValue == nil {
			return
		}
		fields := structValue.GetFields()
		if value, ok := numberField(fields, "vmin"); ok {
			values = append(values, value)
		}
		if value, ok := numberField(fields, "vmax"); ok {
			values = append(values, value)
		}
	}

	for _, item := range pack.GetFacts() {
		fields := item.GetPayload().GetFields()
		addRange(fields["value"].GetStructValue())
		addRange(fields["si"].GetStructValue())
	}
	for _, item := range pack.GetConsensus() {
		fields := item.GetFields()
		if value, ok := numberField(fields, "agreedMin"); ok {
			values = append(values, value)
		}
		if value, ok := numberField(fields, "agreedMax"); ok {
			values = append(values, value)
		}
		for _, source := range fields["sources"].GetListValue().GetValues() {
			addRange(source.GetStructValue())
		}
	}
	return values
}

func containsApprox(values []float64, target float64) bool {
	for _, value := range values {
		if approxEqual(target, value) {
			return true
		}
	}
	return false
}
