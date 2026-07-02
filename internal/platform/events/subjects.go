package events

import (
	"errors"
	"strings"
)

const (
	DocumentRegistered    = "kmap.doc.v1.registered"
	DocumentParsed        = "kmap.doc.v1.parsed"
	DocumentParseFailed   = "kmap.doc.v1.parse-failed"
	DocumentExtracted     = "kmap.doc.v1.extracted"
	FactsCommitted        = "kmap.facts.v1.committed"
	EpistemicClusterDirty = "kmap.epistemic.v1.cluster-dirty"
	EpistemicUpdated      = "kmap.epistemic.v1.updated"
)

const (
	auditPrefix = "kmap.audit.v1."
	dlqPrefix   = "kmap.dlq."
)

func Audit(action string) string {
	return auditPrefix + action
}

func DLQ(stage string) string {
	return dlqPrefix + stage
}

func IsAudit(typ string) bool {
	return strings.HasPrefix(typ, auditPrefix)
}

func IsDLQ(typ string) bool {
	return strings.HasPrefix(typ, dlqPrefix)
}

func ValidateType(typ string) error {
	switch typ {
	case DocumentRegistered, DocumentParsed, DocumentParseFailed, DocumentExtracted,
		FactsCommitted, EpistemicClusterDirty, EpistemicUpdated:
		return nil
	}
	if IsAudit(typ) || IsDLQ(typ) {
		return nil
	}
	return errors.New("unknown event type: " + typ)
}
