package http

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/contracts/openapi"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/audit"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/auth"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/blob"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/nats"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/pg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	maxUploadBytes = 100 << 20
	metadataLimit  = 1 << 20
)

type Server struct {
	search      kmapv1.SearchServiceClient
	ingest      kmapv1.IngestServiceClient
	answer      kmapv1.AnswerServiceClient
	catalog     kmapv1.CatalogServiceClient
	epistemic   kmapv1.EpistemicServiceClient
	blob        blob.Store
	verifier    auth.Verifier
	audit       *audit.Writer
	auditEvents events.Publisher
	auditBus    io.Closer
	logger      *slog.Logger
	pool        *pg.Pool
	rawBucket   string
	corsOrigins []string
	conns       []*grpc.ClientConn
}

type problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail,omitempty"`
	Instance  string `json:"instance,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func NewServer(cfg config.Bundle, logger *slog.Logger) (*Server, error) {
	targets := cfg.Runtime.GRPCClients
	conns := map[string]*grpc.ClientConn{}
	closeAll := func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
	}

	for _, name := range []string{"search", "ingest", "answer", "catalog", "epistemic"} {
		target := targets[name]
		if target == "" {
			closeAll()
			return nil, fmt.Errorf("grpc_clients.%s is required", name)
		}
		conn, err := grpc.NewClient(target,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithChainUnaryInterceptor(auth.UnaryClientInterceptor()),
			grpc.WithChainStreamInterceptor(auth.StreamClientInterceptor()),
		)
		if err != nil {
			closeAll()
			return nil, fmt.Errorf("create %s grpc client: %w", name, err)
		}
		conns[name] = conn
	}

	blobStore, err := blob.New(blob.Config{
		Endpoint:  cfg.Runtime.S3.Endpoint,
		AccessKey: cfg.Runtime.S3.AccessKey,
		SecretKey: cfg.Runtime.S3.SecretKey,
		UseSSL:    cfg.Runtime.S3.UseSSL,
		Region:    cfg.Runtime.S3.Region,
	})
	if err != nil {
		closeAll()
		return nil, fmt.Errorf("create s3 client: %w", err)
	}
	rawBucket := cfg.Runtime.S3.Buckets["raw"]
	if rawBucket == "" {
		closeAll()
		return nil, errors.New("s3.buckets.raw is required")
	}
	if err := blobStore.EnsureBucket(context.Background(), rawBucket); err != nil {
		closeAll()
		return nil, fmt.Errorf("ensure raw bucket: %w", err)
	}

	verifier, err := buildVerifier(cfg.Runtime.Auth)
	if err != nil {
		closeAll()
		return nil, fmt.Errorf("build auth verifier: %w", err)
	}

	pool, err := pg.New(context.Background(), pg.Config{
		DSN:      cfg.Runtime.Postgres.DSN,
		MaxConns: cfg.Runtime.Postgres.MaxConns,
	})
	if err != nil {
		closeAll()
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	var auditEvents events.Publisher
	var auditBus io.Closer
	if url := cfg.Runtime.NATS.URL; url != "" {
		bus, busErr := nats.New(context.Background(), nats.Config{
			URL:     url,
			Name:    "kmap-gateway",
			Streams: []nats.StreamSpec{{Name: "KMAP_AUDIT", Subjects: []string{"kmap.audit.v1.>"}}},
		})
		if busErr != nil {
			logger.Warn("audit event stream disabled", "error", busErr)
		} else {
			auditEvents = bus
			auditBus = bus
		}
	}

	return &Server{
		search:      kmapv1.NewSearchServiceClient(conns["search"]),
		ingest:      kmapv1.NewIngestServiceClient(conns["ingest"]),
		answer:      kmapv1.NewAnswerServiceClient(conns["answer"]),
		catalog:     kmapv1.NewCatalogServiceClient(conns["catalog"]),
		epistemic:   kmapv1.NewEpistemicServiceClient(conns["epistemic"]),
		blob:        blobStore,
		verifier:    verifier,
		audit:       audit.NewWriter(pool.Pool),
		auditEvents: auditEvents,
		auditBus:    auditBus,
		logger:      logger,
		pool:        pool,
		rawBucket:   rawBucket,
		corsOrigins: cfg.Runtime.HTTP.CorsOrigins,
		conns:       []*grpc.ClientConn{conns["search"], conns["ingest"], conns["answer"], conns["catalog"], conns["epistemic"]},
	}, nil
}

func (server *Server) RegisterHTTP(router chi.Router) {
	router.Post("/v1/ask", server.secure(auth.OpAsk, server.askHandler))
	router.Post("/v1/search", server.secure(auth.OpSearch, server.searchHandler))
	router.Get("/v1/entities", server.secure(auth.OpBrowse, server.entitiesHandler))
	router.Get("/v1/entities/{id}", server.secure(auth.OpBrowse, server.entityHandler))
	router.Get("/v1/entities/{id}/facts", server.secure(auth.OpBrowse, server.entityFactsHandler))
	router.Get("/v1/experiments", server.secure(auth.OpBrowse, server.experimentsHandler))
	router.Get("/v1/experts", server.secure(auth.OpBrowse, server.expertsHandler))
	router.Get("/v1/coverage", server.secure(auth.OpBrowse, server.coverageHandler))
	router.Get("/v1/contradictions", server.secure(auth.OpBrowse, server.contradictionsHandler))
	router.Get("/v1/graph", server.secure(auth.OpBrowse, server.graphHandler))
	router.Get("/v1/documents", server.secure(auth.OpBrowse, server.documentsHandler))
	router.Post("/v1/documents", server.secure(auth.OpDocumentUpload, server.uploadDocumentHandler))
	router.Get("/v1/documents/{document_id}/status", server.secure(auth.OpBrowse, server.documentStatusHandler))
	router.Post("/v1/facts/{id}/status", server.secure(auth.OpFactDecision, server.updateFactStatusHandler))
	router.Post("/v1/entities/{id}/merge", server.secure(auth.OpEntityMerge, server.mergeEntityHandler))
	router.Post("/v1/contradictions/{id}/decision", server.secure(auth.OpContradictionDecision, server.decideContradictionHandler))
	router.Get("/openapi.yaml", server.cors(server.openAPIHandler))
	router.Options("/v1/*", server.corsPreflight)
}

func (server *Server) openAPIHandler(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(stdhttp.StatusOK)
	_, _ = w.Write(openapi.Spec)
}

func (server *Server) Close() error {
	var result error
	for _, conn := range server.conns {
		result = errors.Join(result, conn.Close())
	}
	if server.auditBus != nil {
		result = errors.Join(result, server.auditBus.Close())
	}
	if server.pool != nil {
		result = errors.Join(result, server.pool.Close())
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

func (server *Server) uploadDocumentHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	r.Body = stdhttp.MaxBytesReader(w, r.Body, maxUploadBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid multipart request", err.Error())
		return
	}

	upload, meta, err := server.collectParts(r.Context(), reader)
	if err != nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid upload", err.Error())
		return
	}
	if upload == nil {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Missing file part", "")
		return
	}

	request := &kmapv1.RegisterDocumentRequest{
		Title:          titleFromMeta(meta, upload.fileName),
		BlobUri:        upload.blobURI,
		Sha256:         upload.sha256,
		DeclaredMeta:   meta,
		Principal:      principalFromContext(r),
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	}

	resp, err := server.ingest.RegisterDocument(r.Context(), request)
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}

	writeProto(w, resp)
}

func (server *Server) documentStatusHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	documentID := chi.URLParam(r, "document_id")
	if documentID == "" {
		writeProblem(w, r, stdhttp.StatusBadRequest, "invalid_request", "Invalid request", "document_id is required")
		return
	}

	resp, err := server.ingest.GetStatus(r.Context(), &kmapv1.GetStatusRequest{
		Document:  &kmapv1.DocumentRef{DocumentId: documentID},
		Principal: principalFromContext(r),
	})
	if err != nil {
		writeGRPCProblem(w, r, err)
		return
	}

	writeProto(w, resp)
}

type uploadedFile struct {
	blobURI  string
	sha256   []byte
	fileName string
}

func (server *Server) collectParts(ctx context.Context, reader *multipart.Reader) (*uploadedFile, *structpb.Struct, error) {
	var upload *uploadedFile
	meta := map[string]any{}

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		switch part.FormName() {
		case "file":
			uploaded, err := server.streamFile(ctx, part)
			_ = part.Close()
			if err != nil {
				return nil, nil, err
			}
			upload = uploaded
		case "metadata":
			data, err := io.ReadAll(io.LimitReader(part, metadataLimit))
			_ = part.Close()
			if err != nil {
				return nil, nil, err
			}
			if err := json.Unmarshal(data, &meta); err != nil {
				return nil, nil, fmt.Errorf("parse metadata: %w", err)
			}
		default:
			_ = part.Close()
		}
	}

	return upload, metaStruct(meta), nil
}

func (server *Server) streamFile(ctx context.Context, part *multipart.Part) (*uploadedFile, error) {
	key, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate object key: %w", err)
	}
	hasher := sha256.New()
	tee := io.TeeReader(part, hasher)

	blobURI, err := server.blob.Put(ctx, server.rawBucket, key.String(), tee, -1)
	if err != nil {
		return nil, err
	}
	return &uploadedFile{
		blobURI:  blobURI,
		sha256:   hasher.Sum(nil),
		fileName: part.FileName(),
	}, nil
}

func titleFromMeta(meta *structpb.Struct, fallback string) string {
	if value, ok := meta.AsMap()["title"].(string); ok && value != "" {
		return value
	}
	return fallback
}

func metaStruct(meta map[string]any) *structpb.Struct {
	st, err := structpb.NewStruct(meta)
	if err != nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	return st
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
		Type:      "https://kmap.local/problems/" + problemType,
		Title:     title,
		Status:    statusCode,
		Detail:    detail,
		Instance:  r.URL.Path,
		RequestID: r.Header.Get("X-Request-Id"),
	})
}

func writeJSON(w stdhttp.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
