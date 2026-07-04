package http

import (
	"strings"
	"testing"
)

func TestSourceFilename(t *testing.T) {
	t.Parallel()

	cases := []struct {
		title   string
		blobKey string
		want    string
	}{
		{"Доклад Иванова.pdf", "abc123.pdf", "Доклад Иванова.pdf"},
		{"Отчёт без расширения", "abc123.docx", "Отчёт без расширения.docx"},
		{"", "abc123.xlsx", "document.xlsx"},
		{"   ", "abc123", "document"},
	}
	for _, tc := range cases {
		if got := sourceFilename(tc.title, tc.blobKey); got != tc.want {
			t.Errorf("sourceFilename(%q, %q) = %q, want %q", tc.title, tc.blobKey, got, tc.want)
		}
	}
}

func TestContentTypeForOfficeFormats(t *testing.T) {
	t.Parallel()

	zipHead := []byte("PK\x03\x04rest-of-docx-archive")
	if got := contentTypeFor("Доклад.docx", zipHead); got != sourceMIME[".docx"] {
		t.Fatalf("docx content type = %q, want %q", got, sourceMIME[".docx"])
	}
	if got := contentTypeFor("book.xls", nil); got != sourceMIME[".xls"] {
		t.Fatalf("xls content type = %q, want %q", got, sourceMIME[".xls"])
	}
	if strings.Contains(contentTypeFor("report.docx", zipHead), "zip") {
		t.Fatal("docx must not be served as a zip type")
	}
}

func TestContentDispositionEncodesUnicode(t *testing.T) {
	t.Parallel()

	header := contentDisposition("Доклад Иванова.pdf")
	if !strings.HasPrefix(header, "attachment; ") {
		t.Fatalf("expected attachment disposition, got %q", header)
	}
	if !strings.Contains(header, "filename*=UTF-8''") {
		t.Fatalf("expected RFC 5987 filename*, got %q", header)
	}
	if !strings.Contains(header, "%D0%94") {
		t.Fatalf("expected percent-encoded Cyrillic, got %q", header)
	}
	if !strings.Contains(header, ".pdf") {
		t.Fatalf("expected preserved extension, got %q", header)
	}
	if strings.Contains(header, "\"Доклад") {
		t.Fatalf("ascii fallback must not contain raw unicode, got %q", header)
	}
}
