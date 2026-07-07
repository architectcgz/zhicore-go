package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestEngagementCacheBatchGetViewerStatusUsesSingleMGet(t *testing.T) {
	client := &fakeEngagementRedisClient{
		values: map[string]string{
			"content:engagement:user:42:post:post_1": `{"liked":true,"favorited":false}`,
			"content:engagement:user:42:post:post_2": `{"liked":false,"favorited":true}`,
		},
	}
	cache := NewEngagementCache(client, EngagementCacheConfig{TTL: time.Minute})

	got, err := cache.BatchGetViewerStatus(context.Background(), 42, []string{"post_1", "post_2"})
	if err != nil {
		t.Fatalf("BatchGetViewerStatus() error = %v", err)
	}
	if client.mgetCalls != 1 {
		t.Fatalf("MGet calls = %d, want one batch call", client.mgetCalls)
	}
	if len(got) != 2 || !got[0].Liked || !got[1].Favorited {
		t.Fatalf("status = %+v", got)
	}
}

func TestEngagementCacheBatchGetViewerStatusReturnsConfirmedSubsetOnMiss(t *testing.T) {
	client := &fakeEngagementRedisClient{
		values: map[string]string{
			"content:engagement:user:42:post:post_1": `{"liked":true,"favorited":false}`,
		},
	}
	cache := NewEngagementCache(client, EngagementCacheConfig{TTL: time.Minute})

	got, err := cache.BatchGetViewerStatus(context.Background(), 42, []string{"post_1", "post_2"})
	if err != nil {
		t.Fatalf("BatchGetViewerStatus() error = %v", err)
	}
	if len(got) != 1 || got[0].PostID != "post_1" {
		t.Fatalf("status = %+v, want confirmed subset only", got)
	}
}

func TestEngagementCacheStoreMutationWritesKnownViewerState(t *testing.T) {
	client := &fakeEngagementRedisClient{}
	cache := NewEngagementCache(client, EngagementCacheConfig{TTL: 30 * time.Second})

	err := cache.StoreMutation(context.Background(), ports.EngagementMutationRecord{
		PostID:    "post_1",
		ActorID:   42,
		Liked:     true,
		Favorited: false,
	})
	if err != nil {
		t.Fatalf("StoreMutation() error = %v", err)
	}
	if client.msetCalls != 1 {
		t.Fatalf("MSet calls = %d, want 1", client.msetCalls)
	}
	if client.ttl != 30*time.Second {
		t.Fatalf("ttl = %s, want 30s", client.ttl)
	}
	if got := client.values["content:engagement:user:42:post:post_1"]; got != `{"liked":true,"favorited":false}` {
		t.Fatalf("stored value = %s", got)
	}
}

func TestEngagementCacheStoreViewerStatusWritesBatch(t *testing.T) {
	client := &fakeEngagementRedisClient{}
	cache := NewEngagementCache(client, EngagementCacheConfig{TTL: time.Minute})

	err := cache.StoreViewerStatus(context.Background(), 42, []ports.EngagementStatusRecord{
		{PostID: "post_1", Liked: true, Favorited: false},
		{PostID: "post_2", Liked: false, Favorited: true},
	})
	if err != nil {
		t.Fatalf("StoreViewerStatus() error = %v", err)
	}
	if client.msetCalls != 1 || len(client.values) != 2 {
		t.Fatalf("MSet calls=%d values=%v, want one batch write", client.msetCalls, client.values)
	}
}

func TestEngagementCachePropagatesRedisReadError(t *testing.T) {
	client := &fakeEngagementRedisClient{err: errors.New("redis down")}
	cache := NewEngagementCache(client, EngagementCacheConfig{TTL: time.Minute})

	_, err := cache.BatchGetViewerStatus(context.Background(), 42, []string{"post_1"})
	if err == nil {
		t.Fatal("BatchGetViewerStatus() error = nil, want redis error")
	}
}

func TestEngagementCacheNilClientReturnsDependencyError(t *testing.T) {
	cache := NewEngagementCache(nil, EngagementCacheConfig{TTL: time.Minute})

	_, err := cache.BatchGetViewerStatus(context.Background(), 42, []string{"post_1"})
	if err == nil {
		t.Fatal("BatchGetViewerStatus() error = nil, want dependency error")
	}
}

type fakeEngagementRedisClient struct {
	values    map[string]string
	ttl       time.Duration
	err       error
	mgetCalls int
	mgetKeys  []string
	msetCalls int
}

func (f *fakeEngagementRedisClient) MGet(ctx context.Context, keys ...string) ([]string, error) {
	f.mgetCalls++
	f.mgetKeys = append([]string(nil), keys...)
	if f.err != nil {
		return nil, f.err
	}
	values := make([]string, len(keys))
	for i, key := range keys {
		values[i] = f.values[key]
	}
	return values, nil
}

func (f *fakeEngagementRedisClient) MSet(ctx context.Context, values map[string]string, ttl time.Duration) error {
	f.msetCalls++
	f.ttl = ttl
	if f.err != nil {
		return f.err
	}
	if f.values == nil {
		f.values = make(map[string]string, len(values))
	}
	for key, value := range values {
		f.values[key] = value
	}
	return nil
}
