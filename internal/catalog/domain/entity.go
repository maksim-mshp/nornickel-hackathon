package domain

import (
	"errors"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

const (
	StatusActive        = "active"
	StatusPendingReview = "pending_review"
)

var (
	ErrBundleURIRequired = errors.New("bundle_uri is required")
	ErrInvalidBundle     = errors.New("invalid extraction bundle")
	ErrDocumentNotFound  = errors.New("document not found")
	ErrUnknownEntityType = errors.New("unknown entity type")
	ErrUnknownOperator   = errors.New("unknown numeric operator")
)

var validEntityTypes = map[string]struct{}{
	"material": {}, "process": {}, "equipment": {}, "property": {}, "parameter": {},
	"technology": {}, "experiment": {}, "publication": {}, "person": {}, "lab": {},
	"org": {}, "geography": {}, "topic": {}, "economic_indicator": {}, "climate": {}, "facility": {},
}

var validOperators = map[string]struct{}{
	"eq": {}, "lt": {}, "lte": {}, "gt": {}, "gte": {}, "range": {},
	"approx": {}, "from": {}, "to": {}, "pm": {},
}

type Entity struct {
	ID              uuid.UUID
	EType           string
	CanonicalName   string
	CanonicalNameEN string
	Slug            string
	Status          string
	CreatedBy       string
}

type Resolution struct {
	Input      string
	EType      string
	EntityID   *uuid.UUID
	Confidence float64
	Status     string
}

func NewEntity(etype string, name string, nameEN string) (Entity, error) {
	if _, ok := validEntityTypes[etype]; !ok {
		return Entity{}, ErrUnknownEntityType
	}
	canonical := strings.TrimSpace(name)
	if canonical == "" {
		return Entity{}, errors.New("entity name is required")
	}
	return Entity{
		ID:              uuid.Must(uuid.NewV7()),
		EType:           etype,
		CanonicalName:   canonical,
		CanonicalNameEN: strings.TrimSpace(nameEN),
		Slug:            slugify(canonical, etype),
		Status:          StatusPendingReview,
		CreatedBy:       "extract",
	}, nil
}

func ValidateOperator(op string) error {
	if _, ok := validOperators[op]; !ok {
		return ErrUnknownOperator
	}
	return nil
}

func slugify(name string, etype string) string {
	fields := strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	slug := strings.Join(fields, "-")
	return strings.ToLower(etype) + ":" + strings.ToLower(slug)
}
