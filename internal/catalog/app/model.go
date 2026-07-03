package app

type Bundle struct {
	DocumentID       string         `json:"document_id"`
	ExtractorVersion string         `json:"extractor_version"`
	Chunks           []BundleChunk  `json:"chunks"`
	Entities         []BundleEntity `json:"entities"`
	NumericFacts     []BundleFact   `json:"numeric_facts"`
}

type BundleChunk struct {
	Ordinal  int    `json:"ordinal"`
	Text     string `json:"text"`
	Kind     string `json:"kind"`
	PageFrom int    `json:"page_from"`
	PageTo   int    `json:"page_to"`
	Lang     string `json:"lang"`
}

type BundleEntity struct {
	Slug   string `json:"slug"`
	Etype  string `json:"etype"`
	Name   string `json:"name"`
	NameEn string `json:"name_en"`
}

type BundleFact struct {
	SubjectSlug      string            `json:"subject_slug"`
	ParameterSlug    string            `json:"parameter_slug"`
	Operator         string            `json:"operator"`
	ValueRaw         string            `json:"value_raw"`
	Vmin             *float64          `json:"vmin"`
	Vmax             *float64          `json:"vmax"`
	UnitOrig         string            `json:"unit_orig"`
	UnitCode         string            `json:"unit_code"`
	VminSI           *float64          `json:"vmin_si"`
	VmaxSI           *float64          `json:"vmax_si"`
	Conditions       map[string]string `json:"conditions"`
	Quote            string            `json:"quote"`
	Page             int               `json:"page"`
	Geography        string            `json:"geography"`
	ExtractionMethod string            `json:"extraction_method"`
	ExtractorVersion string            `json:"extractor_version"`
	Confidence       float64           `json:"confidence"`
}

type CommitResult struct {
	DocumentID string
	FactIDs    []string
	EntityIDs  []string
}

type Resolution struct {
	Input         string
	EntityID      string
	Slug          string
	CanonicalName string
	Confidence    float64
	Status        string
}
