package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const uriScheme = "s3://"

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Region    string
}

type Store interface {
	EnsureBucket(ctx context.Context, bucket string) error
	Put(ctx context.Context, bucket string, key string, reader io.Reader, size int64) (string, error)
	Get(ctx context.Context, bucket string, key string) (io.ReadCloser, error)
	Exists(ctx context.Context, bucket string, key string) (bool, error)
	URI(bucket string, key string) string
}

func New(cfg Config) (*MinIO, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("s3 endpoint is required")
	}
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	return &MinIO{client: client}, nil
}

type MinIO struct {
	client *minio.Client
}

func (store *MinIO) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := store.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket %s: %w", bucket, err)
	}
	if exists {
		return nil
	}
	if err := store.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: ""}); err != nil {
		return fmt.Errorf("create bucket %s: %w", bucket, err)
	}
	return nil
}

func (store *MinIO) Put(ctx context.Context, bucket string, key string, reader io.Reader, size int64) (string, error) {
	if _, err := store.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}); err != nil {
		return "", fmt.Errorf("put object %s/%s: %w", bucket, key, err)
	}
	return store.URI(bucket, key), nil
}

func (store *MinIO) Exists(ctx context.Context, bucket string, key string) (bool, error) {
	if _, err := store.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{}); err != nil {
		response := minio.ToErrorResponse(err)
		if response.Code == "NoSuchKey" || response.StatusCode == stdhttp.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("stat object %s/%s: %w", bucket, key, err)
	}
	return true, nil
}

func (store *MinIO) Get(ctx context.Context, bucket string, key string) (io.ReadCloser, error) {
	object, err := store.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s/%s: %w", bucket, key, err)
	}
	return object, nil
}

func (store *MinIO) URI(bucket string, key string) string {
	return URI(bucket, key)
}

func URI(bucket string, key string) string {
	return uriScheme + bucket + "/" + key
}

func ParseURI(uri string) (string, string, error) {
	if !strings.HasPrefix(uri, uriScheme) {
		return "", "", fmt.Errorf("invalid blob uri %q: expected %s scheme", uri, uriScheme)
	}
	rest := uri[len(uriScheme):]
	idx := strings.IndexByte(rest, '/')
	if idx <= 0 || idx == len(rest)-1 {
		return "", "", fmt.Errorf("invalid blob uri %q: missing bucket or key", uri)
	}
	return rest[:idx], rest[idx+1:], nil
}
