package app

type entityRef struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type numericValue struct {
	Operator string   `json:"operator"`
	Vmin     *float64 `json:"vmin,omitempty"`
	Vmax     *float64 `json:"vmax,omitempty"`
	Unit     string   `json:"unit"`
}

type provenance struct {
	DocumentID string `json:"documentId"`
	Title      string `json:"title"`
	DocType    string `json:"docType"`
	Page       int    `json:"page"`
	Quote      string `json:"quote"`
	Year       int    `json:"year"`
}

type scoreComponents struct {
	Match      float64 `json:"match"`
	Rerank     float64 `json:"rerank"`
	Source     float64 `json:"source"`
	Validation float64 `json:"validation"`
	Freshness  float64 `json:"freshness"`
}

type fact struct {
	ID               string            `json:"id"`
	Ref              string            `json:"ref"`
	Subject          entityRef         `json:"subject"`
	Parameter        entityRef         `json:"parameter"`
	Value            numericValue      `json:"value"`
	SI               numericValue      `json:"si"`
	Conditions       map[string]string `json:"conditions"`
	Geography        string            `json:"geography"`
	Provenance       provenance        `json:"provenance"`
	ExtractionMethod string            `json:"extractionMethod"`
	ExtractorVersion string            `json:"extractorVersion"`
	Confidence       float64           `json:"confidence"`
	ValidationStatus string            `json:"validationStatus"`
	Score            float64           `json:"score"`
	ScoreComponents  scoreComponents   `json:"scoreComponents"`
}

type consensusSource struct {
	Title     string  `json:"title"`
	Year      int     `json:"year"`
	Geography string  `json:"geography"`
	Vmin      float64 `json:"vmin"`
	Vmax      float64 `json:"vmax"`
}

type consensus struct {
	Parameter    entityRef         `json:"parameter"`
	Unit         string            `json:"unit"`
	Verdict      string            `json:"verdict"`
	AgreedMin    float64           `json:"agreedMin"`
	AgreedMax    float64           `json:"agreedMax"`
	OverlapIndex float64           `json:"overlapIndex"`
	Sources      []consensusSource `json:"sources"`
}

type contradiction struct {
	ID          string   `json:"id"`
	AFactRef    string   `json:"aFactRef"`
	BFactRef    string   `json:"bFactRef"`
	AStatement  string   `json:"aStatement"`
	BStatement  string   `json:"bStatement"`
	Cause       string   `json:"cause"`
	Confounders []string `json:"confounders"`
	Status      string   `json:"status"`
	Confidence  float64  `json:"confidence"`
}

type gapCell struct {
	Label     string   `json:"label"`
	Score     float64  `json:"score"`
	Reasons   []string `json:"reasons"`
	Neighbors []string `json:"neighbors"`
}

type expert struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Lab         string  `json:"lab"`
	Weight      float64 `json:"weight"`
	Reports     int     `json:"reports"`
	Experiments int     `json:"experiments"`
	LastYear    int     `json:"lastYear"`
}

type evidenceStats struct {
	Sources        int `json:"sources"`
	RuSources      int `json:"ruSources"`
	ForeignSources int `json:"foreignSources"`
	YearFrom       int `json:"yearFrom"`
	YearTo         int `json:"yearTo"`
}

type scenario struct {
	slug           string
	intent         string
	materials      []entityRef
	processes      []entityRef
	properties     []entityRef
	facts          []fact
	consensus      []consensus
	contradictions []contradiction
	gaps           []gapCell
	experts        []expert
	stats          evidenceStats
}

func ptr(v float64) *float64 {
	return &v
}
