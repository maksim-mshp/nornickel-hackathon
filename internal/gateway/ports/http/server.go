package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	search kmapv1.SearchServiceClient
	ingest kmapv1.IngestServiceClient
	conns  []*grpc.ClientConn
}

type problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

func NewServer(cfg config.Bundle, _ *slog.Logger) (*Server, error) {
	searchTarget := cfg.Runtime.GRPCClients["search"]
	if searchTarget == "" {
		return nil, errors.New("grpc_clients.search is required")
	}
	ingestTarget := cfg.Runtime.GRPCClients["ingest"]
	if ingestTarget == "" {
		return nil, errors.New("grpc_clients.ingest is required")
	}

	searchConn, err := grpc.NewClient(searchTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create search grpc client: %w", err)
	}
	ingestConn, err := grpc.NewClient(ingestTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = searchConn.Close()
		return nil, fmt.Errorf("create ingest grpc client: %w", err)
	}

	return &Server{
		search: kmapv1.NewSearchServiceClient(searchConn),
		ingest: kmapv1.NewIngestServiceClient(ingestConn),
		conns:  []*grpc.ClientConn{searchConn, ingestConn},
	}, nil
}

func (server *Server) RegisterHTTP(router chi.Router) {
	router.Post("/v1/search", server.searchHandler)
	router.Post("/v1/documents", server.registerDocumentHandler)
}

func (server *Server) Close() error {
	var result error
	for _, conn := range server.conns {
		result = errors.Join(result, conn.Close())
	}
	return result
}

func (server *Server) searchHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req kmapv1.SearchRequest
	body, err := readBody(w, r)
	if err != nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, &req); err != nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	resp, err := server.search.Search(r.Context(), &req)
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}

	writeProto(w, resp)
}

func (server *Server) registerDocumentHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req kmapv1.RegisterDocumentRequest
	body, err := readBody(w, r)
	if err != nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, &req); err != nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	resp, err := server.ingest.RegisterDocument(r.Context(), &req)
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}

	writeProto(w, resp)
}

func readBody(w stdhttp.ResponseWriter, r *stdhttp.Request) ([]byte, error) {
	defer func() {
		_ = r.Body.Close()
	}()
	return io.ReadAll(stdhttp.MaxBytesReader(w, r.Body, 1<<20))
}

func writeProto(w stdhttp.ResponseWriter, protoMsg proto.Message) {
	data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(protoMsg)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, problem{
			Type:   "about:blank",
			Title:  "Internal server error",
			Status: stdhttp.StatusInternalServerError,
			Detail: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(stdhttp.StatusOK)
	_, _ = w.Write(data)
}

func writeGRPCProblem(w stdhttp.ResponseWriter, r *stdhttp.Request, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeProblem(w, r, stdhttp.StatusBadGateway, "upstream_error", "Upstream error", err.Error())
		return
	}
	writeProblem(w, r, grpcHTTPStatus(st.Code()), st.Code().String(), st.Message(), "")
}

func grpcHTTPStatus(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return stdhttp.StatusBadRequest
	case codes.NotFound:
		return stdhttp.StatusNotFound
	case codes.Unauthenticated:
		return stdhttp.StatusUnauthorized
	case codes.PermissionDenied:
		return stdhttp.StatusForbidden
	case codes.Unimplemented:
		return stdhttp.StatusNotImplemented
	case codes.Unavailable:
		return stdhttp.StatusBadGateway
	default:
		return stdhttp.StatusInternalServerError
	}
}

func writeProblem(w stdhttp.ResponseWriter, r *stdhttp.Request, statusCode int, problemType string, title string, detail string) {
	writeJSON(w, statusCode, problem{
		Type:     "https://kmap.local/problems/" + problemType,
		Title:    title,
		Status:   statusCode,
		Detail:   detail,
		Instance: r.URL.Path,
	})
}

func writeJSON(w stdhttp.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
