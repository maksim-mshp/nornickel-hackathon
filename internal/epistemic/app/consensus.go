package app

import (
	"math"
	"sort"
)

const ConsensusEngineVersion = "consensus-1.0"

type ConsensusFact struct {
	Lo         float64
	Hi         float64
	HasRange   bool
	Confidence float64
	DocType    string
	Year       int
	Geography  string
	DocumentID string
	Unit       string
}

type ConsensusResult struct {
	Verdict        string
	AgreedLo       float64
	AgreedHi       float64
	HasRange       bool
	OverlapIndex   float64
	Confidence     float64
	Unit           string
	Sources        int
	RuSources      int
	ForeignSources int
	YearFrom       int
	YearTo         int
}

var consensusSourceReliability = map[string]float64{
	"protocol": 1.0, "report": 1.0, "article": 0.9, "patent": 0.8,
	"handbook": 0.9, "normative": 0.9, "dataset": 1.0, "web": 0.5,
}

func ComputeConsensus(facts []ConsensusFact, currentYear int) ConsensusResult {
	intervals := make([]interval, 0, len(facts))
	documents := map[string]struct{}{}
	result := ConsensusResult{Verdict: "insufficient"}

	for _, fact := range facts {
		if !fact.HasRange {
			continue
		}
		lo, hi := fact.Lo, fact.Hi
		if lo == hi {
			pad := math.Abs(lo) * 0.02
			lo -= pad
			hi += pad
		}
		if hi < lo {
			lo, hi = hi, lo
		}
		weight := fact.Confidence * lookupReliability(fact.DocType) * freshness(currentYear, fact.Year)
		if weight <= 0 {
			weight = 1e-6
		}
		intervals = append(intervals, interval{lo: lo, hi: hi, weight: weight})
		documents[fact.DocumentID] = struct{}{}
		if result.Unit == "" {
			result.Unit = fact.Unit
		}
		countStats(&result, fact)
	}

	result.Sources = len(documents)
	if len(intervals) == 0 {
		return result
	}

	result.AgreedLo = weightedMedian(intervals, func(item interval) float64 { return item.lo })
	result.AgreedHi = weightedMedian(intervals, func(item interval) float64 { return item.hi })
	if result.AgreedHi < result.AgreedLo {
		result.AgreedLo, result.AgreedHi = result.AgreedHi, result.AgreedLo
	}
	result.HasRange = true
	result.OverlapIndex = overlapIndex(intervals)
	result.Confidence = clamp(result.OverlapIndex, 0.3, 0.95)
	result.Verdict = verdict(intervals, len(documents), result)
	return result
}

type interval struct {
	lo     float64
	hi     float64
	weight float64
}

func overlapIndex(intervals []interval) float64 {
	maxLo := intervals[0].lo
	minHi := intervals[0].hi
	minLo := intervals[0].lo
	maxHi := intervals[0].hi
	for _, item := range intervals[1:] {
		maxLo = math.Max(maxLo, item.lo)
		minHi = math.Min(minHi, item.hi)
		minLo = math.Min(minLo, item.lo)
		maxHi = math.Max(maxHi, item.hi)
	}
	union := maxHi - minLo
	if union <= 0 {
		return 1
	}
	inter := math.Max(0, minHi-maxLo)
	return inter / union
}

func verdict(intervals []interval, documents int, result ConsensusResult) string {
	if documents < 2 {
		return "insufficient"
	}
	if result.OverlapIndex >= 0.5 && documents >= 3 {
		return "consensus"
	}
	touching := 0
	for _, item := range intervals {
		if item.hi >= result.AgreedLo && item.lo <= result.AgreedHi {
			touching++
		}
	}
	if float64(touching)/float64(len(intervals)) >= 0.6 {
		return "majority"
	}
	return "split"
}

func weightedMedian(intervals []interval, pick func(interval) float64) float64 {
	type sample struct {
		value  float64
		weight float64
	}
	samples := make([]sample, len(intervals))
	total := 0.0
	for index, item := range intervals {
		samples[index] = sample{value: pick(item), weight: item.weight}
		total += item.weight
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].value < samples[j].value })
	half := total / 2
	cumulative := 0.0
	for _, item := range samples {
		cumulative += item.weight
		if cumulative >= half {
			return item.value
		}
	}
	return samples[len(samples)-1].value
}

func countStats(result *ConsensusResult, fact ConsensusFact) {
	switch fact.Geography {
	case "ru":
		result.RuSources++
	case "foreign":
		result.ForeignSources++
	}
	if fact.Year == 0 {
		return
	}
	if result.YearFrom == 0 || fact.Year < result.YearFrom {
		result.YearFrom = fact.Year
	}
	if fact.Year > result.YearTo {
		result.YearTo = fact.Year
	}
}

func lookupReliability(docType string) float64 {
	if value, ok := consensusSourceReliability[docType]; ok {
		return value
	}
	return 0.7
}

func freshness(currentYear int, docYear int) float64 {
	if docYear == 0 {
		return 0.5
	}
	return math.Exp(-0.1 * float64(currentYear-docYear))
}

func clamp(value float64, low float64, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
