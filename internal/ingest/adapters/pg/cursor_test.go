package pg

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDocumentCursorRoundTrip(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 7, 3, 12, 34, 56, 123456789, time.UTC)
	id := uuid.MustParse("019f2983-6372-7400-8100-a0551d85ca4f")

	encoded := encodeDocumentCursor(when, id)
	if encoded == "" {
		t.Fatal("empty cursor")
	}

	gotTime, gotID, err := decodeDocumentCursor(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !gotTime.Equal(when) {
		t.Errorf("time = %v, want %v", gotTime, when)
	}
	if gotID != id {
		t.Errorf("id = %v, want %v", gotID, id)
	}
}

func TestDocumentCursorRejectsGarbage(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{"not-base64!!", "", "Zm9vYmFy"} {
		if _, _, err := decodeDocumentCursor(bad); err == nil && bad != "" {
			t.Errorf("decodeDocumentCursor(%q) expected error", bad)
		}
	}
}
