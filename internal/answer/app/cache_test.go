package app

import (
	"context"
	"testing"

	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"google.golang.org/grpc"
)

type countingSearch struct {
	pack  *kmapv1.EvidencePack
	calls int
}

func (search *countingSearch) Search(context.Context, *kmapv1.SearchRequest, ...grpc.CallOption) (*kmapv1.SearchResponse, error) {
	search.calls++
	return &kmapv1.SearchResponse{Evidence: search.pack}, nil
}

type memCache struct {
	store map[string]*CachedAnswer
}

func (cache *memCache) Get(_ context.Context, key []byte) (*CachedAnswer, bool, error) {
	value, ok := cache.store[string(key)]
	return value, ok, nil
}

func (cache *memCache) Put(_ context.Context, key []byte, value *CachedAnswer) error {
	cache.store[string(key)] = value
	return nil
}

func collectTypes(t *testing.T, service *Service, question string) []string {
	t.Helper()
	var types []string
	err := service.Ask(context.Background(), &kmapv1.AskRequest{Question: question}, func(event *kmapv1.AskResponse) error {
		types = append(types, event.GetType())
		return nil
	})
	if err != nil {
		t.Fatalf("ask failed: %v", err)
	}
	return types
}

func TestAskServesSecondCallFromCache(t *testing.T) {
	t.Parallel()

	question := "оптимальная скорость циркуляции католита при электроэкстракции никеля?"
	search := &countingSearch{pack: samplecatholytePack(t)}
	cache := &memCache{store: map[string]*CachedAnswer{}}
	service := NewService(search, WithCache(cache))

	first := collectTypes(t, service, question)
	if search.calls != 1 {
		t.Fatalf("expected 1 search call after first ask, got %d", search.calls)
	}
	if len(cache.store) != 1 {
		t.Fatalf("expected answer cached, store size %d", len(cache.store))
	}

	second := collectTypes(t, service, question)
	if search.calls != 1 {
		t.Fatalf("second ask must hit cache, search calls = %d", search.calls)
	}

	if first[0] != "plan" || second[0] != "plan" {
		t.Fatalf("both must start with plan: %v / %v", first, second)
	}
	if first[len(first)-1] != "answer.done" || second[len(second)-1] != "answer.done" {
		t.Fatalf("both must end with answer.done: %v / %v", first, second)
	}
}
