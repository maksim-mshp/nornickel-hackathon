package domain

type Bundle struct {
	Schema            string             `json:"schema"`
	DocumentID        string             `json:"document_id"`
	Version           int                `json:"version"`
	ExtractorVersion  string             `json:"extractor_version"`
	Chunks            []Chunk            `json:"chunks"`
	Entities          []BundleEntity     `json:"entities"`
	NumericCandidates []NumericCandidate `json:"numeric_candidates"`
	NumericFacts      []NumericCandidate `json:"numeric_facts"`
	Quality           map[string]any     `json:"quality"`
}

type Chunk struct {
	ID          string    `json:"id"`
	Text        string    `json:"text"`
	Ordinal     int       `json:"ordinal"`
	PageFrom    int       `json:"page_from"`
	PageTo      int       `json:"page_to"`
	Kind        string    `json:"kind"`
	Lang        string    `json:"lang"`
	SectionPath []string  `json:"section_path"`
	CharFrom    int       `json:"char_from"`
	CharTo      int       `json:"char_to"`
	Embedding   []float32 `json:"embedding"`
}

type BundleEntity struct {
	Type   string `json:"type"`
	EType  string `json:"etype"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	NameEN string `json:"name_en"`
}

type NumericCandidate struct {
	Subject          string         `json:"subject"`
	SubjectSlug      string         `json:"subject_slug"`
	Parameter        string         `json:"parameter"`
	ParameterSlug    string         `json:"parameter_slug"`
	Operator         string         `json:"operator"`
	ValueRaw         string         `json:"value_raw"`
	VMin             *float64       `json:"vmin"`
	VMax             *float64       `json:"vmax"`
	UnitOrig         string         `json:"unit_orig"`
	UnitCode         string         `json:"unit_code"`
	VMinSI           *float64       `json:"vmin_si"`
	VMaxSI           *float64       `json:"vmax_si"`
	Conditions       map[string]any `json:"conditions"`
	ConditionHash    []byte         `json:"condition_hash"`
	Quote            string         `json:"quote"`
	Page             int            `json:"page"`
	CharFrom         int            `json:"char_from"`
	CharTo           int            `json:"char_to"`
	Geography        string         `json:"geography"`
	ExtractionMethod string         `json:"extraction_method"`
	ExtractorVersion string         `json:"extractor_version"`
	Confidence       float32        `json:"confidence"`
	ChunkID          string         `json:"chunk_id"`
	Relation         string         `json:"relation"`
}
