package pg

import (
	"context"
	"testing"
)

func TestNewRejectsEmptyDSN(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for empty dsn")
	}
}

func TestNewRejectsMalformedDSN(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), Config{DSN: "postgres://user bad@/x y z"})
	if err == nil {
		t.Fatal("expected error for malformed dsn")
	}
}

func TestNormalizeRLSDefaults(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		principal  Principal
		wantAccess string
		wantUser   string
	}{
		{name: "empty", principal: Principal{}, wantAccess: "internal", wantUser: "system"},
		{name: "user only", principal: Principal{UserID: "u1"}, wantAccess: "internal", wantUser: "u1"},
		{name: "access only", principal: Principal{DocAccess: "restricted"}, wantAccess: "restricted", wantUser: "system"},
		{name: "both", principal: Principal{UserID: "u2", DocAccess: "public"}, wantAccess: "public", wantUser: "u2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			access, user := normalizeRLS(tc.principal)
			if access != tc.wantAccess || user != tc.wantUser {
				t.Fatalf("normalizeRLS(%+v) = (%q,%q), want (%q,%q)", tc.principal, access, user, tc.wantAccess, tc.wantUser)
			}
		})
	}
}
