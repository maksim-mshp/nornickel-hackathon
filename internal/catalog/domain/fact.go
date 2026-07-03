package domain

import (
	"math"
	"strings"

	"github.com/google/uuid"
)

const (
	MethodDeterministic = "deterministic"
	MethodHybrid        = "hybrid"
	MethodLLM           = "llm"
	MethodCatalog       = "catalog"

	FactMachineExtracted = "machine_extracted"
	FactWeakEvidence     = "weak_evidence"
	FactNeedsUnitReview  = "needs_unit_review"
	FactRejected         = "rejected"
)

type NumericFact struct {
	ID               uuid.UUID
	SubjectID        uuid.UUID
	ParameterID      uuid.UUID
	ChunkID          *uuid.UUID
	Relation         string
	Operator         string
	ValueRaw         string
	VMin             *float64
	VMax             *float64
	UnitOrig         string
	UnitCode         string
	VMinSI           *float64
	VMaxSI           *float64
	Conditions       map[string]any
	ConditionHash    []byte
	Quote            string
	Page             int
	CharFrom         int
	CharTo           int
	Geography        string
	ExtractionMethod string
	ExtractorVersion string
	Confidence       float32
	ValidationStatus string
}

type ParameterDef struct {
	PlausibleMin *float64
	PlausibleMax *float64
}

func (d ParameterDef) plausible(vmin *float64, vmax *float64) bool {
	const rel = 1e-9
	for _, value := range []*float64{vmin, vmax} {
		if value == nil {
			continue
		}
		if d.PlausibleMin != nil && *value < *d.PlausibleMin-(math.Abs(*d.PlausibleMin)*rel+rel) {
			return false
		}
		if d.PlausibleMax != nil && *value > *d.PlausibleMax+(math.Abs(*d.PlausibleMax)*rel+rel) {
			return false
		}
	}
	return true
}

func ClassifyFact(fact NumericFact, def *ParameterDef) (string, float32) {
	confidence := fact.Confidence
	if strings.TrimSpace(fact.UnitCode) == "" {
		return FactNeedsUnitReview, confidence
	}
	if !validBounds(fact) {
		return FactRejected, confidence
	}
	if def != nil && !def.plausible(fact.VMinSI, fact.VMaxSI) {
		if confidence > 0.5 {
			confidence = 0.5
		}
		return FactWeakEvidence, confidence
	}
	return FactMachineExtracted, confidence
}

func validBounds(fact NumericFact) bool {
	switch fact.Operator {
	case "range", "pm":
		if fact.VMin == nil || fact.VMax == nil {
			return false
		}
	case "lte", "lt", "to":
		if fact.VMax == nil {
			return false
		}
	case "gte", "gt", "from":
		if fact.VMin == nil {
			return false
		}
	case "eq", "approx":
		if fact.VMin == nil && fact.VMax == nil {
			return false
		}
	}
	if fact.VMinSI != nil && fact.VMaxSI != nil && *fact.VMinSI > *fact.VMaxSI {
		return false
	}
	return true
}
