package domain

import (
	"errors"
	"testing"
)

func TestAllowedFetchURL(t *testing.T) {
	t.Parallel()
	allow := []string{"arxiv.org", "elibrary.ru"}

	cases := []struct {
		url  string
		want error
	}{
		{"https://arxiv.org/abs/2401.00001", nil},
		{"https://export.arxiv.org/pdf/x", nil},
		{"https://elibrary.ru/item.asp?id=1", nil},
		{"http://arxiv.org/abs/x", ErrURLScheme},
		{"https://evil.com/arxiv.org", ErrURLBlocked},
		{"https://notarxiv.org/x", ErrURLBlocked},
		{"https://arxiv.org.evil.com/x", ErrURLBlocked},
	}
	for _, tc := range cases {
		err := AllowedFetchURL(tc.url, allow)
		if tc.want == nil && err != nil {
			t.Errorf("AllowedFetchURL(%q) = %v, want nil", tc.url, err)
		}
		if tc.want != nil && !errors.Is(err, tc.want) {
			t.Errorf("AllowedFetchURL(%q) = %v, want %v", tc.url, err, tc.want)
		}
	}
}

func TestAllowedFetchURLEmptyAllowlistBlocks(t *testing.T) {
	t.Parallel()
	if err := AllowedFetchURL("https://arxiv.org/x", nil); !errors.Is(err, ErrURLBlocked) {
		t.Errorf("empty allowlist must block (fail-closed), got %v", err)
	}
}
