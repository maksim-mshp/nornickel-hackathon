package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
)

type OIDCConfig struct {
	Issuer         string
	Audience       string
	DocAccessClaim string
}

type OIDCVerifier struct {
	config         OIDCConfig
	docAccessClaim string
	mu             sync.Mutex
	verifier       *oidc.IDTokenVerifier
}

func NewOIDCVerifier(config OIDCConfig) (*OIDCVerifier, error) {
	if config.Issuer == "" {
		return nil, fmt.Errorf("oidc issuer is required")
	}
	docAccessClaim := config.DocAccessClaim
	if docAccessClaim == "" {
		docAccessClaim = "doc_access"
	}
	return &OIDCVerifier{config: config, docAccessClaim: docAccessClaim}, nil
}

func (verifier *OIDCVerifier) ensure(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	verifier.mu.Lock()
	defer verifier.mu.Unlock()
	if verifier.verifier != nil {
		return verifier.verifier, nil
	}
	provider, err := oidc.NewProvider(ctx, verifier.config.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	verifier.verifier = provider.Verifier(&oidc.Config{
		ClientID:          verifier.config.Audience,
		SkipClientIDCheck: verifier.config.Audience == "",
	})
	return verifier.verifier, nil
}

func (verifier *OIDCVerifier) Verify(ctx context.Context, token string) (Principal, error) {
	inner, err := verifier.ensure(ctx)
	if err != nil {
		return Principal{}, ErrUnauthorized
	}
	idToken, err := inner.Verify(ctx, token)
	if err != nil {
		return Principal{}, ErrUnauthorized
	}
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return Principal{}, ErrUnauthorized
	}
	roles := KnownRoles(realmRoles(claims))
	docAccess := stringClaim(claims, verifier.docAccessClaim)
	if docAccess == "" {
		docAccess = DocAccessForRoles(roles)
	}
	return Principal{
		UserID:    idToken.Subject,
		Name:      stringClaim(claims, "name"),
		Roles:     roles,
		DocAccess: docAccess,
	}, nil
}

func realmRoles(claims map[string]any) []string {
	realmAccess, ok := claims["realm_access"].(map[string]any)
	if !ok {
		return nil
	}
	raw, ok := realmAccess["roles"].([]any)
	if !ok {
		return nil
	}
	roles := make([]string, 0, len(raw))
	for _, item := range raw {
		if role, ok := item.(string); ok {
			roles = append(roles, role)
		}
	}
	return roles
}

func stringClaim(claims map[string]any, key string) string {
	if value, ok := claims[key].(string); ok {
		return value
	}
	return ""
}
