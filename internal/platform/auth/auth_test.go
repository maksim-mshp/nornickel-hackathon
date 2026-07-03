package auth

import (
	"context"
	"testing"
)

func TestDocAccessForRoles(t *testing.T) {
	cases := []struct {
		roles []string
		want  string
	}{
		{[]string{RolePartner}, AccessPublic},
		{[]string{RoleResearcher}, AccessInternal},
		{[]string{RoleAnalyst}, AccessInternal},
		{[]string{RoleManager}, AccessConfidential},
		{[]string{RoleExpert}, AccessConfidential},
		{[]string{RoleAdmin}, AccessRestricted},
		{[]string{RoleResearcher, RoleAdmin}, AccessRestricted},
		{nil, AccessPublic},
	}
	for _, testCase := range cases {
		if got := DocAccessForRoles(testCase.roles); got != testCase.want {
			t.Errorf("DocAccessForRoles(%v) = %q, want %q", testCase.roles, got, testCase.want)
		}
	}
}

func TestAllowed(t *testing.T) {
	researcher := Principal{Roles: []string{RoleResearcher}}
	expert := Principal{Roles: []string{RoleExpert}}
	partner := Principal{Roles: []string{RolePartner}}

	if !Allowed(OpAsk, researcher) {
		t.Error("researcher must be allowed to ask")
	}
	if Allowed(OpContradictionDecision, researcher) {
		t.Error("researcher must not decide contradictions")
	}
	if !Allowed(OpContradictionDecision, expert) {
		t.Error("expert must decide contradictions")
	}
	if !Allowed(OpAsk, partner) {
		t.Error("partner must be allowed to ask")
	}
	if Allowed(OpSearch, partner) {
		t.Error("partner must not use raw search")
	}
}

func TestDemoVerifier(t *testing.T) {
	verifier := NewDemoVerifier(map[string]DemoToken{
		"tok-expert": {Sub: "expert-1", Name: "Эксперт", Roles: []string{RoleExpert, "offline_access"}},
	})

	principal, err := verifier.Verify(context.Background(), "tok-expert")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if principal.UserID != "expert-1" {
		t.Errorf("user id = %q", principal.UserID)
	}
	if len(principal.Roles) != 1 || principal.Roles[0] != RoleExpert {
		t.Errorf("roles = %v, unknown roles must be dropped", principal.Roles)
	}
	if principal.DocAccess != AccessConfidential {
		t.Errorf("doc access = %q, want confidential", principal.DocAccess)
	}
	if _, err := verifier.Verify(context.Background(), "nope"); err == nil {
		t.Error("unknown token must fail")
	}
}
