package auth

import "context"

type DemoToken struct {
	Sub       string
	Name      string
	Roles     []string
	DocAccess string
}

type DemoVerifier struct {
	principals map[string]Principal
}

func NewDemoVerifier(tokens map[string]DemoToken) *DemoVerifier {
	principals := make(map[string]Principal, len(tokens))
	for token, definition := range tokens {
		roles := KnownRoles(definition.Roles)
		docAccess := definition.DocAccess
		if docAccess == "" {
			docAccess = DocAccessForRoles(roles)
		}
		principals[token] = Principal{
			UserID:    definition.Sub,
			Name:      definition.Name,
			Roles:     roles,
			DocAccess: docAccess,
		}
	}
	return &DemoVerifier{principals: principals}
}

func (verifier *DemoVerifier) Verify(_ context.Context, token string) (Principal, error) {
	principal, ok := verifier.principals[token]
	if !ok {
		return Principal{}, ErrUnauthorized
	}
	return principal, nil
}
