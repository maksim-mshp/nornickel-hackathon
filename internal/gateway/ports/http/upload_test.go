package http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/blob"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeIngestClient struct {
	gotRequest *kmapv1.RegisterDocumentRequest
	response   *kmapv1.RegisterDocumentResponse
	err        error
}

func (client *fakeIngestClient) RegisterDocument(_ context.Context, req *kmapv1.RegisterDocumentRequest, _ ...grpc.CallOption) (*kmapv1.RegisterDocumentResponse, error) {
	client.gotRequest = req
	if client.err != nil {
		return nil, client.err
	}
	return client.response, nil
}

func (client *fakeIngestClient) GetStatus(_ context.Context, _ *kmapv1.GetStatusRequest, _ ...grpc.CallOption) (*kmapv1.GetStatusResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not used")
}

func testServer(ingest *fakeIngestClient, blobStore blob.Store) *Server {
	return &Server{ingest: ingest, blob: blobStore, rawBucket: "kmap-raw"}
}

func newMultipartUpload(t *testing.T, fileName string, payload []byte, metadata map[string]any) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if metadata != nil {
		field, err := writer.CreateFormField("metadata")
		if err != nil {
			t.Fatalf("create metadata field: %v", err)
		}
		_ = json.NewEncoder(field).Encode(metadata)
	}

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/documents", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestUploadDocumentStreamsToBlobAndRegisters(t *testing.T) {
	t.Parallel()

	payload := []byte("document-content")
	ingest := &fakeIngestClient{response: &kmapv1.RegisterDocumentResponse{
		DocumentId: "0197-doc", Version: 1, Status: "registered",
	}}
	blobStore := blob.NewMemStore()
	server := testServer(ingest, blobStore)

	req := newMultipartUpload(t, "report.pdf", payload, map[string]any{
		"title":   "Ni electrowinning",
		"doc_type": "report",
	})
	rec := httptest.NewRecorder()
	server.uploadDocumentHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	sum := sha256.Sum256(payload)
	if !bytes.Equal(ingest.gotRequest.GetSha256(), sum[:]) {
		t.Fatalf("expected sha256 %x, got %x", sum[:], ingest.gotRequest.GetSha256())
	}
	if ingest.gotRequest.GetBlobUri() == "" {
		t.Fatal("expected blob uri")
	}
	if !strings.HasPrefix(ingest.gotRequest.GetBlobUri(), "s3://kmap-raw/") {
		t.Fatalf("unexpected blob uri: %s", ingest.gotRequest.GetBlobUri())
	}
	if ingest.gotRequest.GetTitle() != "Ni electrowinning" {
		t.Fatalf("expected title from metadata, got %q", ingest.gotRequest.GetTitle())
	}
	if ingest.gotRequest.GetPrincipal().GetUserId() != "demo" {
		t.Fatalf("expected demo principal, got %q", ingest.gotRequest.GetPrincipal().GetUserId())
	}

	bucket, key, err := blob.ParseURI(ingest.gotRequest.GetBlobUri())
	if err != nil {
		t.Fatalf("parse uri: %v", err)
	}
	reader, err := blobStore.Get(req.Context(), bucket, key)
	if err != nil {
		t.Fatalf("get blob: %v", err)
	}
	defer func() { _ = reader.Close() }()
}

func TestUploadDocumentRejectsMissingFile(t *testing.T) {
	t.Parallel()

	ingest := &fakeIngestClient{response: &kmapv1.RegisterDocumentResponse{}}
	server := testServer(ingest, blob.NewMemStore())

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	field, _ := writer.CreateFormField("metadata")
	_ = json.NewEncoder(field).Encode(map[string]any{"title": "x"})
	_ = writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/v1/documents", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	server.uploadDocumentHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if ingest.gotRequest != nil {
		t.Fatal("expected ingest not to be called")
	}
}

func TestUploadDocumentPropagatesIngestError(t *testing.T) {
	t.Parallel()

	ingest := &fakeIngestClient{err: status.Error(codes.InvalidArgument, "bad sha")}
	server := testServer(ingest, blob.NewMemStore())

	req := newMultipartUpload(t, "report.pdf", []byte("x"), nil)
	rec := httptest.NewRecorder()
	server.uploadDocumentHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
