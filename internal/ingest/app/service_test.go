package app

import (
	"context"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRegisterDocumentRequiresSHA256(t *testing.T) {
	t.Parallel()

	service := NewService()
	_, err := service.RegisterDocument(context.Background(), &kmapv1.RegisterDocumentRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestRegisterDocumentAndGetStatus(t *testing.T) {
	t.Parallel()

	service := NewService()
	registered, err := service.RegisterDocument(context.Background(), &kmapv1.RegisterDocumentRequest{
		Sha256: []byte("hash"),
	})
	if err != nil {
		t.Fatalf("expected register to succeed: %v", err)
	}
	if registered.GetDocumentId() == "" {
		t.Fatal("expected document id")
	}
	if registered.GetVersion() != 1 {
		t.Fatalf("expected version 1, got %d", registered.GetVersion())
	}

	got, err := service.GetStatus(context.Background(), &kmapv1.GetStatusRequest{
		Document: &kmapv1.DocumentRef{DocumentId: registered.GetDocumentId()},
	})
	if err != nil {
		t.Fatalf("expected status to succeed: %v", err)
	}
	if got.GetStatus() != "registered" {
		t.Fatalf("expected registered status, got %q", got.GetStatus())
	}
	if len(got.GetStages()) == 0 {
		t.Fatal("expected stages")
	}
}

func TestGetStatusNotFound(t *testing.T) {
	t.Parallel()

	service := NewService()
	_, err := service.GetStatus(context.Background(), &kmapv1.GetStatusRequest{
		Document: &kmapv1.DocumentRef{DocumentId: "missing"},
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}
