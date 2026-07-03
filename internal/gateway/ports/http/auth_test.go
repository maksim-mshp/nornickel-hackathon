package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/auth"
)

func secureTestServer() *Server {
	return &Server{
		verifier: auth.NewDemoVerifier(map[string]auth.DemoToken{
			"tok-researcher": {Sub: "researcher-1", Roles: []string{auth.RoleResearcher}},
			"tok-expert":     {Sub: "expert-1", Roles: []string{auth.RoleExpert}},
		}),
	}
}

func TestSecureRejectsMissingToken(t *testing.T) {
	t.Parallel()
	server := secureTestServer()
	handler := server.secure(auth.OpBrowse, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/v1/entities", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestSecureRejectsInvalidToken(t *testing.T) {
	t.Parallel()
	server := secureTestServer()
	handler := server.secure(auth.OpBrowse, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/entities", nil)
	req.Header.Set("Authorization", "Bearer nope")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestSecureForbidsRoleWithoutPermission(t *testing.T) {
	t.Parallel()
	server := secureTestServer()
	handler := server.secure(auth.OpContradictionDecision, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/contradictions/c1/decision", nil)
	req.Header.Set("Authorization", "Bearer tok-researcher")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestSecureAllowsPermittedRole(t *testing.T) {
	t.Parallel()
	server := secureTestServer()
	var seen string
	handler := server.secure(auth.OpBrowse, func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.FromContext(r.Context())
		seen = principal.UserID
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/entities", nil)
	req.Header.Set("Authorization", "Bearer tok-researcher")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if seen != "researcher-1" {
		t.Fatalf("expected principal in context, got %q", seen)
	}
}
