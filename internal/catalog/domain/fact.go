package domain

import "github.com/google/uuid"

const (
	MethodDeterministic = "deterministic"
	MethodHybrid        = "hybrid"
	MethodLLM           = "llm"
	MethodCatalog       = "catalog"
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
}
