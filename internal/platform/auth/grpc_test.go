package auth

import (
	"context"
	"strconv"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
)

func freshTimestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func signedMetadata(key []byte, principal Principal, timestamp string) metadata.MD {
	md := metadata.MD{}
	md.Set(mdUser, principal.UserID)
	md.Set(mdDocAccess, principal.DocAccess)
	if len(principal.Roles) > 0 {
		md.Set(mdRoles, principal.Roles...)
	}
	md.Set(mdTimestamp, timestamp)
	md.Set(mdSignature, signPrincipal(key, principal, timestamp))
	return md
}

func TestPrincipalFromIncomingAcceptsValidSignature(t *testing.T) {
	t.Parallel()
	key := []byte("shared-secret")
	principal := Principal{UserID: "researcher", Roles: []string{"researcher"}, DocAccess: "internal"}
	ctx := metadata.NewIncomingContext(context.Background(), signedMetadata(key, principal, freshTimestamp()))

	got, ok := principalFromIncoming(ctx, key)
	if !ok {
		t.Fatal("valid signed principal rejected")
	}
	if got.DocAccess != "internal" || got.UserID != "researcher" {
		t.Fatalf("unexpected principal: %+v", got)
	}
}

func TestPrincipalFromIncomingRejectsUnsignedForgery(t *testing.T) {
	t.Parallel()
	key := []byte("shared-secret")
	forged := metadata.MD{}
	forged.Set(mdUser, "attacker")
	forged.Set(mdDocAccess, AccessRestricted)
	forged.Set(mdRoles, "admin")
	ctx := metadata.NewIncomingContext(context.Background(), forged)

	if _, ok := principalFromIncoming(ctx, key); ok {
		t.Fatal("forged unsigned principal must be rejected when signing key is set")
	}
}

func TestPrincipalFromIncomingRejectsTamperedDocAccess(t *testing.T) {
	t.Parallel()
	key := []byte("shared-secret")
	principal := Principal{UserID: "researcher", Roles: []string{"researcher"}, DocAccess: "internal"}
	md := signedMetadata(key, principal, freshTimestamp())
	md.Set(mdDocAccess, AccessRestricted)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	if _, ok := principalFromIncoming(ctx, key); ok {
		t.Fatal("tampered doc_access must invalidate the signature")
	}
}

func TestPrincipalFromIncomingRejectsWrongKey(t *testing.T) {
	t.Parallel()
	principal := Principal{UserID: "researcher", Roles: []string{"researcher"}, DocAccess: "internal"}
	md := signedMetadata([]byte("attacker-key"), principal, freshTimestamp())
	ctx := metadata.NewIncomingContext(context.Background(), md)

	if _, ok := principalFromIncoming(ctx, []byte("shared-secret")); ok {
		t.Fatal("signature from a different key must be rejected")
	}
}

func TestPrincipalFromIncomingRejectsStaleTimestamp(t *testing.T) {
	t.Parallel()
	key := []byte("shared-secret")
	principal := Principal{UserID: "researcher", Roles: []string{"researcher"}, DocAccess: "internal"}
	stale := strconv.FormatInt(time.Now().Add(-2*maxPrincipalAge).Unix(), 10)
	ctx := metadata.NewIncomingContext(context.Background(), signedMetadata(key, principal, stale))

	if _, ok := principalFromIncoming(ctx, key); ok {
		t.Fatal("stale principal must be rejected to limit replay")
	}
}
