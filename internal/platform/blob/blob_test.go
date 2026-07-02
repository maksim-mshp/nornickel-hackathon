package blob

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestURIAndParseRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		bucket string
		key    string
	}{
		{"kmap-raw", "0197-doc"},
		{"kmap-bundles", "a/b/c/bundle.json"},
	}
	for _, tc := range cases {
		uri := URI(tc.bucket, tc.key)
		if uri != "s3://"+tc.bucket+"/"+tc.key {
			t.Fatalf("unexpected uri %q", uri)
		}
		bucket, key, err := ParseURI(uri)
		if err != nil {
			t.Fatalf("parse uri: %v", err)
		}
		if bucket != tc.bucket || key != tc.key {
			t.Fatalf("parse mismatch: got %s/%s", bucket, key)
		}
	}
}

func TestParseURIRejectsInvalid(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"",
		"http://kmap-raw/x",
		"s3://nokey/",
		"s3://",
		"s3://bucketonly",
	}
	for _, uri := range invalid {
		if _, _, err := ParseURI(uri); err == nil {
			t.Fatalf("expected error for %q", uri)
		}
	}
}

func TestMemStorePutGetRoundTrip(t *testing.T) {
	t.Parallel()

	store := NewMemStore()
	ctx := t.Context()
	payload := []byte("document-bytes")

	if err := store.EnsureBucket(ctx, "kmap-raw"); err != nil {
		t.Fatalf("ensure bucket: %v", err)
	}

	uri, err := store.Put(ctx, "kmap-raw", "doc-1", bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	bucket, key, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("parse uri: %v", err)
	}
	if bucket != "kmap-raw" || key != "doc-1" {
		t.Fatalf("unexpected uri parts: %s/%s", bucket, key)
	}

	reader, err := store.Get(ctx, bucket, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = reader.Close() }()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload mismatch: got %q want %q", got, payload)
	}
}

func TestMemStoreGetMissing(t *testing.T) {
	t.Parallel()

	store := NewMemStore()
	_, err := store.Get(t.Context(), "kmap-raw", "missing")
	if err == nil {
		t.Fatal("expected error for missing object")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}
