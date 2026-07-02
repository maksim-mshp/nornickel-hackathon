package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewBuildsValidEnvelope(t *testing.T) {
	t.Parallel()

	data := map[string]any{"document_id": "0197", "sha256": "abcd"}
	env, err := New(Event{
		Type:    DocumentRegistered,
		Source:  "kmap/ingest",
		Subject: "0197",
		Data:    data,
	})
	if err != nil {
		t.Fatalf("expected new to succeed: %v", err)
	}

	if env.SpecVersion != SpecVersion {
		t.Fatalf("expected specversion %q, got %q", SpecVersion, env.SpecVersion)
	}
	if env.ID == "" {
		t.Fatal("expected envelope id")
	}
	if env.DataContentType != ContentType {
		t.Fatalf("expected content type %q, got %q", ContentType, env.DataContentType)
	}
	if env.Time.IsZero() {
		t.Fatal("expected non-zero time")
	}
	if env.Type != DocumentRegistered {
		t.Fatalf("expected type %q, got %q", DocumentRegistered, env.Type)
	}

	var decoded map[string]any
	if err := env.UnmarshalData(&decoded); err != nil {
		t.Fatalf("expected data to unmarshal: %v", err)
	}
	if decoded["document_id"] != "0197" {
		t.Fatalf("unexpected data: %v", decoded)
	}
}

func TestNewAllowsNilData(t *testing.T) {
	t.Parallel()

	env, err := New(Event{Type: EpistemicUpdated, Source: "kmap/epistemic"})
	if err != nil {
		t.Fatalf("expected new to succeed: %v", err)
	}
	if env.Data != nil {
		t.Fatalf("expected nil data, got %s", env.Data)
	}
	if err := env.Validate(); err != nil {
		t.Fatalf("expected valid envelope: %v", err)
	}
}

func TestNewRejectsUnknownType(t *testing.T) {
	t.Parallel()

	_, err := New(Event{Type: "kmap.bogus.v1.x", Source: "kmap/ingest"})
	if err == nil {
		t.Fatal("expected validation error for unknown type")
	}
}

func TestNewRejectsEmptySource(t *testing.T) {
	t.Parallel()

	_, err := New(Event{Type: DocumentParsed, Source: ""})
	if err == nil {
		t.Fatal("expected error for empty source")
	}
}

func TestNewRejectsOversizedPayload(t *testing.T) {
	t.Parallel()

	oversized := make([]byte, MaxPayloadBytes+1)
	for i := range oversized {
		oversized[i] = 'a'
	}

	_, err := New(Event{Type: DocumentParsed, Source: "kmap/parse", Data: map[string]any{"blob": string(oversized)}})
	if err == nil {
		t.Fatal("expected error for oversized payload")
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	t.Parallel()

	env, err := New(Event{Type: FactsCommitted, Source: "kmap/catalog", Subject: "doc-1", Data: map[string]any{"n": 3}})
	if err != nil {
		t.Fatalf("expected new to succeed: %v", err)
	}

	raw, err := env.Marshal()
	if err != nil {
		t.Fatalf("expected marshal to succeed: %v", err)
	}

	decoded, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("expected unmarshal to succeed: %v", err)
	}
	if decoded.ID != env.ID || decoded.Type != env.Type || decoded.Source != env.Source {
		t.Fatalf("roundtrip mismatch: %+v vs %+v", decoded, env)
	}
}

func TestUnmarshalRejectsInvalid(t *testing.T) {
	t.Parallel()

	if _, err := Unmarshal([]byte("{not json")); err == nil {
		t.Fatal("expected error for invalid json")
	}

	env, err := New(Event{Type: DocumentParsed, Source: "kmap/parse"})
	if err != nil {
		t.Fatalf("expected new to succeed: %v", err)
	}
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("expected marshal to succeed: %v", err)
	}
	if _, err := Unmarshal(raw); err != nil {
		t.Fatalf("expected valid envelope to pass: %v", err)
	}
}

func TestEnvelopeValidateRejectsBadSpecVersion(t *testing.T) {
	t.Parallel()

	env := Envelope{
		SpecVersion: "0.3",
		ID:          "id",
		Source:      "kmap/x",
		Type:        DocumentParsed,
		Time:        time.Now().UTC(),
	}
	if err := env.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEnvelopeValidateRequiresDataContentType(t *testing.T) {
	t.Parallel()

	env := Envelope{
		SpecVersion: SpecVersion,
		ID:          "id",
		Source:      "kmap/x",
		Type:        DocumentParsed,
		Time:        time.Now().UTC(),
		Data:        json.RawMessage(`{}`),
	}
	err := env.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing datacontenttype")
	}
}
