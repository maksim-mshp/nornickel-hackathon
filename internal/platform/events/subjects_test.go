package events

import "testing"

func TestValidateTypeAcceptsKnownSubjects(t *testing.T) {
	t.Parallel()

	known := []string{
		DocumentRegistered,
		DocumentParsed,
		DocumentParseFailed,
		DocumentExtracted,
		FactsCommitted,
		EpistemicClusterDirty,
		EpistemicUpdated,
	}
	for _, typ := range known {
		if err := ValidateType(typ); err != nil {
			t.Fatalf("expected %q to be valid: %v", typ, err)
		}
	}
}

func TestValidateTypeAcceptsAuditAndDLQPrefixes(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{Audit("search"), Audit("export"), DLQ("parse"), DLQ("extract")} {
		if err := ValidateType(typ); err != nil {
			t.Fatalf("expected %q to be valid: %v", typ, err)
		}
	}
}

func TestAuditAndDLQClassifiers(t *testing.T) {
	t.Parallel()

	if !IsAudit(Audit("search")) {
		t.Fatal("expected audit classifier to match audit subject")
	}
	if IsAudit(DLQ("parse")) {
		t.Fatal("expected audit classifier to reject dlq subject")
	}
	if !IsDLQ(DLQ("parse")) {
		t.Fatal("expected dlq classifier to match dlq subject")
	}
	if IsDLQ(Audit("search")) {
		t.Fatal("expected dlq classifier to reject audit subject")
	}
}

func TestValidateTypeRejectsUnknown(t *testing.T) {
	t.Parallel()

	if err := ValidateType("kmap.bogus.v1.x"); err == nil {
		t.Fatal("expected error for unknown type")
	}
	if err := ValidateType("kmap.audit.v2.x"); err == nil {
		t.Fatal("expected error for wrong audit version")
	}
}
