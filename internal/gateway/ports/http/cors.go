package http

import (
	stdhttp "net/http"
)

func (server *Server) cors(handler stdhttp.HandlerFunc) stdhttp.HandlerFunc {
	return func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		server.applyCORS(w, r)
		handler(w, r)
	}
}

func (server *Server) corsPreflight(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	server.applyCORS(w, r)
	w.WriteHeader(stdhttp.StatusNoContent)
}

func (server *Server) applyCORS(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	origin := server.allowedOrigin(r.Header.Get("Origin"))
	if origin == "" {
		return
	}
	header := w.Header()
	header.Set("Access-Control-Allow-Origin", origin)
	header.Set("Vary", "Origin")
	header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	header.Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, X-Request-Id, Authorization")
}

func (server *Server) allowedOrigin(origin string) string {
	if origin == "" {
		return ""
	}
	for _, allowed := range server.corsOrigins {
		if allowed == "*" {
			return "*"
		}
		if allowed == origin {
			return origin
		}
	}
	return ""
}
