package domain

import "testing"

func ptr(value float64) *float64 { return &value }

func TestClassifyFactMissingUnit(t *testing.T) {
	t.Parallel()
	fact := NumericFact{Operator: "eq", UnitCode: "", VMin: ptr(60), VMax: ptr(60), Confidence: 0.9}
	status, confidence := ClassifyFact(fact, nil)
	if status != FactNeedsUnitReview {
		t.Fatalf("expected needs_unit_review, got %s", status)
	}
	if confidence != 0.9 {
		t.Fatalf("expected confidence unchanged, got %v", confidence)
	}
}

func TestClassifyFactInvertedRangeRejected(t *testing.T) {
	t.Parallel()
	fact := NumericFact{Operator: "range", UnitCode: "celsius", VMin: ptr(80), VMax: ptr(60), VMinSI: ptr(353.15), VMaxSI: ptr(333.15), Confidence: 0.9}
	if status, _ := ClassifyFact(fact, nil); status != FactRejected {
		t.Fatalf("expected rejected for inverted range, got %s", status)
	}
}

func TestClassifyFactMissingBoundRejected(t *testing.T) {
	t.Parallel()
	fact := NumericFact{Operator: "range", UnitCode: "celsius", VMin: ptr(60), Confidence: 0.9}
	if status, _ := ClassifyFact(fact, nil); status != FactRejected {
		t.Fatalf("expected rejected for range without vmax, got %s", status)
	}
}

func TestClassifyFactImplausibleWeakEvidence(t *testing.T) {
	t.Parallel()
	def := &ParameterDef{PlausibleMin: ptr(173), PlausibleMax: ptr(2300)}
	fact := NumericFact{Operator: "eq", UnitCode: "celsius", VMin: ptr(5000), VMax: ptr(5000), VMinSI: ptr(5273.15), VMaxSI: ptr(5273.15), Confidence: 0.97}
	status, confidence := ClassifyFact(fact, def)
	if status != FactWeakEvidence {
		t.Fatalf("expected weak_evidence for implausible value, got %s", status)
	}
	if confidence > 0.5 {
		t.Fatalf("expected confidence lowered to <=0.5, got %v", confidence)
	}
}

func TestClassifyFactPlausibleBoundaryMachineExtracted(t *testing.T) {
	t.Parallel()
	def := &ParameterDef{PlausibleMin: ptr(213.15), PlausibleMax: ptr(1973.15)}
	fact := NumericFact{Operator: "eq", UnitCode: "celsius", VMin: ptr(-60), VMax: ptr(-60), VMinSI: ptr(213.15), VMaxSI: ptr(213.15), Confidence: 0.97}
	if status, _ := ClassifyFact(fact, def); status != FactMachineExtracted {
		t.Fatalf("expected machine_extracted at plausible boundary, got %s", status)
	}
}
