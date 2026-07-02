package blob

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

type MemStore struct {
	mu      sync.Mutex
	buckets map[string]map[string][]byte
}

func NewMemStore() *MemStore {
	return &MemStore{buckets: map[string]map[string][]byte{}}
}

func (store *MemStore) EnsureBucket(_ context.Context, bucket string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, ok := store.buckets[bucket]; !ok {
		store.buckets[bucket] = map[string][]byte{}
	}
	return nil
}

func (store *MemStore) Put(_ context.Context, bucket string, key string, reader io.Reader, _ int64) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read blob: %w", err)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, ok := store.buckets[bucket]; !ok {
		store.buckets[bucket] = map[string][]byte{}
	}
	store.buckets[bucket][key] = data
	return store.URI(bucket, key), nil
}

func (store *MemStore) Get(_ context.Context, bucket string, key string) (io.ReadCloser, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	objects, ok := store.buckets[bucket]
	if !ok {
		return nil, fmt.Errorf("bucket %s not found", bucket)
	}
	data, ok := objects[key]
	if !ok {
		return nil, fmt.Errorf("object %s/%s not found", bucket, key)
	}
	return io.NopCloser(bytes.NewReader(append([]byte(nil), data...))), nil
}

func (store *MemStore) URI(bucket string, key string) string {
	return URI(bucket, key)
}
