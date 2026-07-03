package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

type OIDCConfig struct {
	Issuer         string
	Audience       string
	DocAccessClaim string
}

type OIDCVerifier struct {
	verifier       *oidc.IDTokenVerifier
	docAccessClaim string
}

func NewOIDCVerifier(ctx context.Context, config OIDCConfig) (*OIDCVerifier, error) {
	if config.Issuer == "" {
		return nil, fmt.Errorf("oidc issuer is required")
	}
	provider, err := oidc.NewProvider(ctx, config.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	oidcConfig := &oidc.Config{ClientID: config.Audience, SkipClientIDCheck: config.Audience == ""}
	docAccessClaim := config.DocAccessClaim
	if docAccessClaim == "" {
		docAccessClaim = "doc_access"
	}
	return &OIDCVerifier{
		verifier:       provider.Verifier(oidcConfig),
		docAccessClaim: docAccessClaim,
	}, nil
}

func (verifier *OIDCVerifier) Verify(ctx context.Context, token string) (Principal, error) {
	idToken, err := verifier.verifier.Verify(ctx, token)
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
