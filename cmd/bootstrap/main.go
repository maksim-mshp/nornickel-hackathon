package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/auth"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/blob"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

const principalID = "bootstrap"

var yearPattern = regexp.MustCompile(`(?:19|20)\d{2}`)

func main() {
	if err := run(); err != nil {
		slog.Error("bootstrap failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	configRoot := flag.String("config", "configs", "configuration root")
	env := flag.String("env", "dev", "environment")
	flag.Parse()

	cfg, err := config.Load(*configRoot, *env, "bootstrap")
	if err != nil {
		return err
	}
	logger := newLogger(cfg.Runtime.Log)

	settings := cfg.Runtime.Bootstrap
	if settings.CorpusDir == "" {
		return errors.New("bootstrap.corpus_dir is required")
	}
	if settings.Concurrency <= 0 {
		settings.Concurrency = 4
	}
	maxBytes := int64(settings.MaxFileMB) << 20

	rawBucket := cfg.Runtime.S3.Buckets["raw"]
	if rawBucket == "" {
		return errors.New("s3.buckets.raw is required")
	}
	ingestTarget := cfg.Runtime.GRPCClients["ingest"]
	if ingestTarget == "" {
		return errors.New("grpc_clients.ingest is required")
	}

	store, err := blob.New(blob.Config{
		Endpoint:  cfg.Runtime.S3.Endpoint,
		AccessKey: cfg.Runtime.S3.AccessKey,
		SecretKey: cfg.Runtime.S3.SecretKey,
		UseSSL:    cfg.Runtime.S3.UseSSL,
		Region:    cfg.Runtime.S3.Region,
	})
	if err != nil {
		return fmt.Errorf("create s3 client: %w", err)
	}

	ctx := auth.WithPrincipal(context.Background(), auth.Principal{
		UserID:    principalID,
		Roles:     []string{auth.RoleAdmin},
		DocAccess: auth.AccessRestricted,
	})

	if err := store.EnsureBucket(ctx, rawBucket); err != nil {
		return fmt.Errorf("ensure raw bucket: %w", err)
	}

	conn, err := grpc.NewClient(ingestTarget,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(auth.UnaryClientInterceptor([]byte(cfg.Runtime.Auth.SigningKey))),
	)
	if err != nil {
		return fmt.Errorf("create ingest client: %w", err)
	}
	defer func() { _ = conn.Close() }()
	client := kmapv1.NewIngestServiceClient(conn)

	files, err := collectFiles(settings.CorpusDir, extensionSet(settings.IncludeExtensions))
	if err != nil {
		return fmt.Errorf("scan corpus: %w", err)
	}
	logger.Info("corpus scan complete", "dir", settings.CorpusDir, "candidates", len(files))

	importer := &importer{
		store:     store,
		client:    client,
		bucket:    rawBucket,
		corpusDir: settings.CorpusDir,
		settings:  settings,
		maxBytes:  maxBytes,
		logger:    logger,
	}
	importer.runAll(ctx, files)

	logger.Info("bootstrap complete",
		"registered", importer.registered.Load(),
		"duplicate", importer.duplicate.Load(),
		"skipped", importer.skipped.Load(),
		"failed", importer.failed.Load(),
	)
	if importer.failed.Load() > 0 {
		return fmt.Errorf("%d files failed to import", importer.failed.Load())
	}
	return nil
}

type importer struct {
	store     blob.Store
	client    kmapv1.IngestServiceClient
	bucket    string
	corpusDir string
	settings  config.Bootstrap
	maxBytes  int64
	logger    *slog.Logger

	registered atomic.Int64
	duplicate  atomic.Int64
	skipped    atomic.Int64
	failed     atomic.Int64
}

func (imp *importer) runAll(ctx context.Context, files []string) {
	semaphore := make(chan struct{}, imp.settings.Concurrency)
	var wg sync.WaitGroup
	for _, path := range files {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(path string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			imp.importFile(ctx, path)
		}(path)
	}
	wg.Wait()
}

func (imp *importer) importFile(ctx context.Context, path string) {
	info, err := os.Stat(path)
	if err != nil {
		imp.logger.Warn("stat failed", "path", path, "error", err)
		imp.skipped.Add(1)
		return
	}
	if imp.maxBytes > 0 && info.Size() > imp.maxBytes {
		imp.logger.Warn("file too large, skipping", "path", path, "bytes", info.Size())
		imp.skipped.Add(1)
		return
	}

	sum, err := hashFile(path)
	if err != nil {
		imp.logger.Error("hash failed", "path", path, "error", err)
		imp.failed.Add(1)
		return
	}
	key := hex.EncodeToString(sum) + strings.ToLower(filepath.Ext(path))
	blobURI := imp.store.URI(imp.bucket, key)

	exists, err := imp.store.Exists(ctx, imp.bucket, key)
	if err != nil {
		imp.logger.Error("blob stat failed", "path", path, "error", err)
		imp.failed.Add(1)
		return
	}
	if !exists {
		if err := imp.upload(ctx, path, key, info.Size()); err != nil {
			imp.logger.Error("upload failed", "path", path, "error", err)
			imp.failed.Add(1)
			return
		}
	}

	rel, err := filepath.Rel(imp.corpusDir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	resp, err := imp.client.RegisterDocument(ctx, &kmapv1.RegisterDocumentRequest{
		Title:        filepath.Base(path),
		BlobUri:      blobURI,
		Sha256:       sum,
		DeclaredMeta: declaredMeta(imp.settings, path, rel),
		Principal: &kmapv1.Principal{
			UserId:    principalID,
			Roles:     []string{auth.RoleAdmin},
			DocAccess: auth.AccessRestricted,
		},
	})
	if err != nil {
		imp.logger.Error("register failed", "path", rel, "error", err)
		imp.failed.Add(1)
		return
	}
	if resp.GetDuplicate() {
		imp.duplicate.Add(1)
		return
	}
	imp.registered.Add(1)
	imp.logger.Info("registered", "path", rel, "document_id", resp.GetDocumentId())
}

func (imp *importer) upload(ctx context.Context, path string, key string, size int64) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	if _, err := imp.store.Put(ctx, imp.bucket, key, file, size); err != nil {
		return err
	}
	return nil
}

func collectFiles(root string, extensions map[string]struct{}) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !included(path, extensions) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func included(path string, extensions map[string]struct{}) bool {
	if len(extensions) == 0 {
		return true
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	_, ok := extensions[ext]
	return ok
}

func extensionSet(list []string) map[string]struct{} {
	set := make(map[string]struct{}, len(list))
	for _, ext := range list {
		set[strings.TrimPrefix(strings.ToLower(ext), ".")] = struct{}{}
	}
	return set
}

func declaredMeta(settings config.Bootstrap, path string, rel string) *structpb.Struct {
	fields := map[string]any{
		"doc_type":     docType(path),
		"geography":    settings.Geography,
		"access_level": settings.AccessLevel,
		"source_path":  rel,
	}
	if year := detectYear(rel); year > 0 {
		fields["year"] = year
	}
	value, err := structpb.NewStruct(fields)
	if err != nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	return value
}

func docType(path string) string {
	switch strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".") {
	case "xls", "xlsx", "csv":
		return "dataset"
	case "htm", "html":
		return "web"
	default:
		return "report"
	}
}

func detectYear(rel string) int {
	match := yearPattern.FindString(rel)
	if match == "" {
		return 0
	}
	year := 0
	for _, digit := range match {
		year = year*10 + int(digit-'0')
	}
	return year
}

func hashFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}

func newLogger(cfg config.Log) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	if strings.EqualFold(cfg.Format, "text") {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
