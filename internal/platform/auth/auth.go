package auth

import (
	"context"
	"errors"
	"slices"
)

const (
	RoleResearcher = "researcher"
	RoleAnalyst    = "analyst"
	RoleManager    = "manager"
	RoleExpert     = "expert"
	RoleAdmin      = "admin"
	RolePartner    = "partner"
)

const (
	AccessPublic       = "public"
	AccessInternal     = "internal"
	AccessConfidential = "confidential"
	AccessRestricted   = "restricted"
)

var ErrUnauthorized = errors.New("unauthorized")

var knownRoles = map[string]struct{}{
	RoleResearcher: {}, RoleAnalyst: {}, RoleManager: {},
	RoleExpert: {}, RoleAdmin: {}, RolePartner: {},
}

var docAccessByRole = map[string]int{
	RolePartner: 0, RoleResearcher: 1, RoleAnalyst: 1,
	RoleManager: 2, RoleExpert: 2, RoleAdmin: 3,
}

var docAccessLevels = []string{AccessPublic, AccessInternal, AccessConfidential, AccessRestricted}

type Principal struct {
	UserID    string
	Name      string
	Roles     []string
	DocAccess string
}

func (principal Principal) HasRole(role string) bool {
	return slices.Contains(principal.Roles, role)
}

func (principal Principal) HasAnyRole(roles ...string) bool {
	return slices.ContainsFunc(roles, principal.HasRole)
}

func KnownRoles(roles []string) []string {
	result := make([]string, 0, len(roles))
	for _, role := range roles {
		if _, ok := knownRoles[role]; ok {
			result = append(result, role)
		}
	}
	return result
}

func DocAccessForRoles(roles []string) string {
	highest := 0
	for _, role := range roles {
		if level, ok := docAccessByRole[role]; ok && level > highest {
			highest = level
		}
	}
	return docAccessLevels[highest]
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func FromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok
}
