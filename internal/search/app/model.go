package app

type EntityRef struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type NumericValue struct {
	Operator string   `json:"operator"`
	Vmin     *float64 `json:"vmin,omitempty"`
	Vmax     *float64 `json:"vmax,omitempty"`
	Unit     string   `json:"unit"`
}

type Provenance struct {
	DocumentID string `json:"documentId"`
	Title      string `json:"title"`
	DocType    string `json:"docType"`
	Page       int    `json:"page"`
	Quote      string `json:"quote"`
	Year       int    `json:"year"`
}

type ScoreComponents struct {
	Match      float64 `json:"match"`
	Rerank     float64 `json:"rerank"`
	Source     float64 `json:"source"`
	Validation float64 `json:"validation"`
	Freshness  float64 `json:"freshness"`
}

type Fact struct {
	ID               string            `json:"id"`
	Ref              string            `json:"ref"`
	Subject          EntityRef         `json:"subject"`
	Parameter        EntityRef         `json:"parameter"`
	Value            NumericValue      `json:"value"`
	SI               NumericValue      `json:"si"`
	Conditions       map[string]string `json:"conditions"`
	Geography        string            `json:"geography"`
	Provenance       Provenance        `json:"provenance"`
	ExtractionMethod string            `json:"extractionMethod"`
	ExtractorVersion string            `json:"extractorVersion"`
	Confidence       float64           `json:"confidence"`
	ValidationStatus string            `json:"validationStatus"`
	Score            float64           `json:"score"`
	ScoreComponents  ScoreComponents   `json:"scoreComponents"`
}

type ConsensusSource struct {
	Title     string  `json:"title"`
	Year      int     `json:"year"`
	Geography string  `json:"geography"`
	Vmin      float64 `json:"vmin"`
	Vmax      float64 `json:"vmax"`
}

type Consensus struct {
	Parameter    EntityRef         `json:"parameter"`
	Unit         string            `json:"unit"`
	Verdict      string            `json:"verdict"`
	AgreedMin    float64           `json:"agreedMin"`
	AgreedMax    float64           `json:"agreedMax"`
	OverlapIndex float64           `json:"overlapIndex"`
	Sources      []ConsensusSource `json:"sources"`
}

type Contradiction struct {
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

type GapCell struct {
	Label     string   `json:"label"`
	Score     float64  `json:"score"`
	Reasons   []string `json:"reasons"`
	Neighbors []string `json:"neighbors"`
}

type Expert struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Lab         string  `json:"lab"`
	Weight      float64 `json:"weight"`
	Reports     int     `json:"reports"`
	Experiments int     `json:"experiments"`
	LastYear    int     `json:"lastYear"`
}

type EvidenceStats struct {
	Sources        int `json:"sources"`
	RuSources      int `json:"ruSources"`
	ForeignSources int `json:"foreignSources"`
	YearFrom       int `json:"yearFrom"`
	YearTo         int `json:"yearTo"`
}

type EvidencePack struct {
	Facts          []Fact
	Consensus      []Consensus
	Contradictions []Contradiction
	Gaps           []GapCell
	Experts        []Expert
	GraphNodes     []GraphNode
	GraphEdges     []GraphEdge
	Stats          EvidenceStats
}

type GraphNode struct {
	ID    string
	Type  string
	Label string
}

type GraphEdge struct {
	ID            string
	Src           string
	Dst           string
	Rel           string
	Weight        float64
	Confidence    float64
	Contradiction bool
}
