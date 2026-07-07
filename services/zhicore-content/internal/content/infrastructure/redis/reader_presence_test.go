package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestReaderPresenceStoreUpsertWritesSessionTTLAndReturnsCount(t *testing.T) {
	client := &fakePresenceRedisClient{count: 3}
	store := NewReaderPresenceStore(client)

	got, err := store.UpsertReaderSession(context.Background(), ports.ReaderSessionInput{
		PostID:    "post_1",
		SessionID: "sess_1",
		UserID:    42,
		TTL:       30 * time.Second,
	})
	if err != nil {
		t.Fatalf("UpsertReaderSession() error = %v", err)
	}
	if client.setKey != "content:presence:post:post_1:session:sess_1" || client.setValue != "42" || client.ttl != 30*time.Second {
		t.Fatalf("set key=%s value=%s ttl=%s", client.setKey, client.setValue, client.ttl)
	}
	if client.countPrefix != "content:presence:post:post_1:session:" || got.OnlineCount != 3 || got.TTL != 30*time.Second {
		t.Fatalf("result=%+v countPrefix=%s", got, client.countPrefix)
	}
}

func TestReaderPresenceStoreDeleteIsBestEffortAdapterOperation(t *testing.T) {
	client := &fakePresenceRedisClient{}
	store := NewReaderPresenceStore(client)

	err := store.DeleteReaderSession(context.Background(), ports.ReaderSessionInput{PostID: "post_1", SessionID: "sess_1"})
	if err != nil {
		t.Fatalf("DeleteReaderSession() error = %v", err)
	}
	if len(client.deletedKeys) != 1 || client.deletedKeys[0] != "content:presence:post:post_1:session:sess_1" {
		t.Fatalf("deleted keys = %#v", client.deletedKeys)
	}
}

func TestReaderPresenceStoreGetCountsPostSessions(t *testing.T) {
	client := &fakePresenceRedisClient{count: 5}
	store := NewReaderPresenceStore(client)

	got, err := store.GetReaderPresence(context.Background(), "post_1")
	if err != nil {
		t.Fatalf("GetReaderPresence() error = %v", err)
	}
	if client.countPrefix != "content:presence:post:post_1:session:" || got.OnlineCount != 5 {
		t.Fatalf("result=%+v countPrefix=%s", got, client.countPrefix)
	}
}

func TestReaderPresenceStorePropagatesRedisError(t *testing.T) {
	client := &fakePresenceRedisClient{err: errors.New("redis down")}
	store := NewReaderPresenceStore(client)

	_, err := store.GetReaderPresence(context.Background(), "post_1")
	if err == nil {
		t.Fatal("GetReaderPresence() error = nil, want redis error")
	}
}

type fakePresenceRedisClient struct {
	setKey      string
	setValue    string
	ttl         time.Duration
	countPrefix string
	count       int
	deletedKeys []string
	err         error
}

func (f *fakePresenceRedisClient) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	f.setKey = key
	f.setValue = value
	f.ttl = ttl
	return f.err
}

func (f *fakePresenceRedisClient) Del(ctx context.Context, keys ...string) error {
	f.deletedKeys = append([]string(nil), keys...)
	return f.err
}

func (f *fakePresenceRedisClient) CountPrefix(ctx context.Context, prefix string) (int, error) {
	f.countPrefix = prefix
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}
