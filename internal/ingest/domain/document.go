package domain

import (
	"errors"

	"github.com/google/uuid"
)

const (
	StatusRegistered = "registered"
)

type Document struct {
	ID          uuid.UUID
	Title       string
	DocType     string
	Lang        string
	Year        int
	Geography   string
	AccessLevel string
	SourceURI   string
	SHA256      []byte
	Status      string
	Version     int
	UploadedBy  string
	Meta        map[string]any
	BlobURI     string
}

type Stage struct {
	Stage   string
	Status  string
	Attempt int
	Error   string
}

func DefaultStages() []Stage {
	return []Stage{
		{Stage: "register", Status: "done"},
		{Stage: "parse", Status: "pending"},
		{Stage: "extract", Status: "pending"},
		{Stage: "commit", Status: "pending"},
		{Stage: "epistemic", Status: "pending"},
	}
}

var (
	ErrSHA256Required     = errors.New("sha256 is required")
	ErrBlobURIRequired    = errors.New("blob_uri is required")
	ErrDocumentNotFound   = errors.New("document not found")
	ErrInvalidDocType     = errors.New("invalid doc_type")
	ErrInvalidGeography   = errors.New("invalid geography")
	ErrInvalidAccessLevel = errors.New("invalid access_level")
)

var (
	validDocTypes = map[string]struct{}{
		"article": {}, "report": {}, "patent": {}, "protocol": {},
		"handbook": {}, "normative": {}, "dataset": {}, "web": {},
	}
	validGeographies = map[string]struct{}{
		"ru": {}, "foreign": {}, "global": {}, "unknown": {},
	}
	validAccessLevels = map[string]struct{}{
		"public": {}, "internal": {}, "confidential": {}, "restricted": {},
	}
)

type Meta struct {
	DocType     string
	Geography   string
	AccessLevel string
}

func NormalizeMeta(docType string, geography string, accessLevel string) (Meta, error) {
	if docType == "" {
		docType = "report"
	}
	if geography == "" {
		geography = "unknown"
	}
	if accessLevel == "" {
		accessLevel = "internal"
	}
	if _, ok := validDocTypes[docType]; !ok {
		return Meta{}, ErrInvalidDocType
	}
	if _, ok := validGeographies[geography]; !ok {
		return Meta{}, ErrInvalidGeography
	}
	if _, ok := validAccessLevels[accessLevel]; !ok {
		return Meta{}, ErrInvalidAccessLevel
	}
	return Meta{DocType: docType, Geography: geography, AccessLevel: accessLevel}, nil
}
