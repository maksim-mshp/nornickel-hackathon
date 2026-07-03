package app

import "testing"

func numericFact(lo float64, hi float64, doc string, year int) ConsensusFact {
	return ConsensusFact{
		Lo: lo, Hi: hi, HasRange: true, Confidence: 0.9,
		DocType: "report", Year: year, DocumentID: doc, Unit: "m/s",
	}
}

func TestConsensusInsufficientBelowTwoDocuments(t *testing.T) {
	t.Parallel()
	result := ComputeConsensus([]ConsensusFact{numericFact(0.8, 1.0, "d1", 2024)}, 2026)
	if result.Verdict != "insufficient" {
		t.Fatalf("want insufficient, got %s", result.Verdict)
	}
	if result.Sources != 1 {
		t.Fatalf("want 1 source, got %d", result.Sources)
	}
}

func TestConsensusOnOverlappingRanges(t *testing.T) {
	t.Parallel()
	facts := []ConsensusFact{
		numericFact(0.80, 1.00, "d1", 2024),
		numericFact(0.85, 1.05, "d2", 2023),
		numericFact(0.82, 0.98, "d3", 2025),
	}
	result := ComputeConsensus(facts, 2026)
	if result.Verdict != "consensus" {
		t.Fatalf("want consensus, got %s (oi=%v)", result.Verdict, result.OverlapIndex)
	}
	if !result.HasRange || result.AgreedHi < result.AgreedLo {
		t.Fatalf("bad agreed range %v..%v", result.AgreedLo, result.AgreedHi)
	}
	if result.OverlapIndex < 0.5 {
		t.Fatalf("want overlap >= 0.5, got %v", result.OverlapIndex)
	}
}

func TestConsensusSplitOnDisjointRanges(t *testing.T) {
	t.Parallel()
	facts := []ConsensusFact{
		numericFact(0.10, 0.20, "d1", 2024),
		numericFact(0.80, 0.90, "d2", 2023),
	}
	result := ComputeConsensus(facts, 2026)
	if result.Verdict != "split" {
		t.Fatalf("want split, got %s (oi=%v)", result.Verdict, result.OverlapIndex)
	}
	if result.OverlapIndex != 0 {
		t.Fatalf("disjoint ranges must not overlap, got %v", result.OverlapIndex)
	}
}

func TestConsensusIgnoresUnboundedFacts(t *testing.T) {
	t.Parallel()
	facts := []ConsensusFact{
		{HasRange: false, DocumentID: "d1"},
		{HasRange: false, DocumentID: "d2"},
	}
	result := ComputeConsensus(facts, 2026)
	if result.HasRange {
		t.Fatal("no numeric facts, want no agreed range")
	}
	if result.Verdict != "insufficient" {
		t.Fatalf("want insufficient, got %s", result.Verdict)
	}
}
