package domain

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrURLScheme  = errors.New("only https urls are allowed for fetch")
	ErrURLBlocked = errors.New("url host is not in the fetch allowlist")
)

func AllowedFetchURL(rawURL string, allowedDomains []string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	if parsed.Scheme != "https" {
		return ErrURLScheme
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return ErrURLBlocked
	}
	for _, domain := range allowedDomains {
		allowed := strings.ToLower(strings.TrimSpace(domain))
		if allowed == "" {
			continue
		}
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return nil
		}
	}
	return ErrURLBlocked
}
