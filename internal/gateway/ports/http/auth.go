package http

import (
	"context"
	"fmt"
	"net"
	stdhttp "net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/audit"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/auth"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

type auditMeta struct {
	action     string
	objectType string
	persist    bool
}

var auditActions = map[auth.Operation]auditMeta{
	auth.OpAsk:                   {action: "search.ask", objectType: "query"},
	auth.OpSearch:                {action: "search.query", objectType: "query"},
	auth.OpDocumentUpload:        {action: "document.upload", objectType: "document", persist: true},
	auth.OpFactDecision:          {action: "fact.decision", objectType: "fact", persist: true},
	auth.OpEntityMerge:           {action: "entity.merge", objectType: "entity", persist: true},
	auth.OpContradictionDecision: {action: "contradiction.decision", objectType: "contradiction", persist: true},
}

func buildVerifier(cfg config.Auth) (auth.Verifier, error) {
	switch cfg.Mode {
	case "", "demo":
		return demoVerifier(cfg), nil
	case "oidc":
		return oidcVerifier(cfg)
	case "hybrid":
		oidc, err := oidcVerifier(cfg)
		if err != nil {
			return nil, err
		}
		return auth.NewCompositeVerifier(demoVerifier(cfg), oidc), nil
	default:
		return nil, fmt.Errorf("unsupported auth mode %q", cfg.Mode)
	}
}

func demoVerifier(cfg config.Auth) *auth.DemoVerifier {
	tokens := make(map[string]auth.DemoToken, len(cfg.Demo.Tokens))
	for token, definition := range cfg.Demo.Tokens {
		tokens[token] = auth.DemoToken{
			Sub:       definition.Sub,
			Name:      definition.Name,
			Roles:     definition.Roles,
			DocAccess: definition.DocAccess,
		}
	}
	return auth.NewDemoVerifier(tokens)
}

func oidcVerifier(cfg config.Auth) (*auth.OIDCVerifier, error) {
	return auth.NewOIDCVerifier(auth.OIDCConfig{
		Issuer:         cfg.OIDC.Issuer,
		Audience:       cfg.OIDC.Audience,
		DocAccessClaim: cfg.OIDC.DocAccessClaim,
	})
}

func (server *Server) secure(operation auth.Operation, next stdhttp.HandlerFunc) stdhttp.HandlerFunc {
	return server.cors(server.authorize(operation, next))
}

func (server *Server) authorize(operation auth.Operation, next stdhttp.HandlerFunc) stdhttp.HandlerFunc {
	return func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		token := bearerToken(r)
		if token == "" {
			writeProblem(w, r, stdhttp.StatusUnauthorized, "unauthorized", "Authentication required", "missing bearer token")
			return
		}
		principal, err := server.verifier.Verify(r.Context(), token)
		if err != nil {
			writeProblem(w, r, stdhttp.StatusUnauthorized, "unauthorized", "Authentication required", "invalid token")
			return
		}
		if !auth.Allowed(operation, principal) {
			writeProblem(w, r, stdhttp.StatusForbidden, "forbidden", "Access denied", "role not permitted for this operation")
			return
		}

		ctx := auth.WithPrincipal(r.Context(), principal)
		meta, audited := auditActions[operation]
		if !audited {
			next(w, r.WithContext(ctx))
			return
		}

		recorder := &statusRecorder{ResponseWriter: w, status: stdhttp.StatusOK}
		next(recorder, r.WithContext(ctx))
		if recorder.status >= 200 && recorder.status < 300 {
			server.emitAuditEvent(ctx, principal, meta, chi.URLParam(r, "id"))
			if meta.persist {
				server.recordAudit(ctx, r, principal, meta)
			}
		}
	}
}

func (server *Server) emitAuditEvent(ctx context.Context, principal auth.Principal, meta auditMeta, objectID string) {
	if server.auditEvents == nil {
		return
	}
	envelope, err := events.New(events.Event{
		Type:    events.Audit(meta.action),
		Source:  "kmap/gateway",
		Subject: principal.UserID,
		Data: map[string]any{
			"actor_id":    principal.UserID,
			"action":      meta.action,
			"object_type": meta.objectType,
			"object_id":   objectID,
			"roles":       principal.Roles,
		},
	})
	if err != nil {
		server.logger.Warn("build audit event failed", "error", err, "action", meta.action)
		return
	}
	if err := server.auditEvents.Publish(ctx, envelope); err != nil {
		server.logger.Warn("publish audit event failed", "error", err, "action", meta.action)
	}
}

func (server *Server) recordAudit(ctx context.Context, r *stdhttp.Request, principal auth.Principal, meta auditMeta) {
	if server.audit == nil {
		return
	}
	err := server.audit.Write(ctx, audit.Record{
		ActorID:    principal.UserID,
		Action:     meta.action,
		ObjectType: meta.objectType,
		ObjectID:   chi.URLParam(r, "id"),
		RequestID:  r.Header.Get("X-Request-Id"),
		IP:         clientIP(r),
		Details:    map[string]any{"roles": principal.Roles},
	})
	if err != nil {
		server.logger.Warn("audit write failed", "error", err, "action", meta.action)
	}
}

func principalFromContext(r *stdhttp.Request) *kmapv1.Principal {
	principal, ok := auth.FromContext(r.Context())
	if !ok {
		return &kmapv1.Principal{UserId: "anonymous", DocAccess: auth.AccessPublic}
	}
	return &kmapv1.Principal{
		UserId:    principal.UserID,
		Roles:     principal.Roles,
		DocAccess: principal.DocAccess,
	}
}

func bearerToken(r *stdhttp.Request) string {
	const prefix = "bearer "
	header := r.Header.Get("Authorization")
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return ""
}

func clientIP(r *stdhttp.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if net.ParseIP(host) == nil {
		return ""
	}
	return host
}

type statusRecorder struct {
	stdhttp.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(code int) {
	recorder.status = code
	recorder.ResponseWriter.WriteHeader(code)
}

func (recorder *statusRecorder) Flush() {
	if flusher, ok := recorder.ResponseWriter.(stdhttp.Flusher); ok {
		flusher.Flush()
	}
}
