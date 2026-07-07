package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestUpsertReaderSessionAnonymousIsNoopSuccess(t *testing.T) {
	deps := newCreatePostDeps()
	service := NewService(deps.asDeps())

	got, err := service.UpsertReaderSession(context.Background(), ReaderSessionCommand{PostID: "post_1", SessionID: "sess_1"})
	if err != nil {
		t.Fatalf("UpsertReaderSession() error = %v", err)
	}
	if deps.presence.upsertCalls != 0 {
		t.Fatalf("presence upsert calls = %d, want 0", deps.presence.upsertCalls)
	}
	if got.PostID != "post_1" || !got.Degraded || got.OnlineCount != 0 || got.TTLSeconds != 30 {
		t.Fatalf("result = %+v, want anonymous degraded no-op", got)
	}
}

func TestUpsertReaderSessionWritesLoggedInPresence(t *testing.T) {
	deps := newCreatePostDeps()
	deps.presence.upsertResult = ports.ReaderPresenceRecord{PostID: "post_1", OnlineCount: 3, TTL: 30 * time.Second}
	service := NewService(deps.asDeps())

	got, err := service.UpsertReaderSession(context.Background(), ReaderSessionCommand{
		Actor:     &Actor{UserID: 42},
		PostID:    "post_1",
		SessionID: "sess_1",
	})
	if err != nil {
		t.Fatalf("UpsertReaderSession() error = %v", err)
	}
	if deps.presence.upsertCalls != 1 || deps.presence.upsertInput.UserID != 42 || deps.presence.upsertInput.SessionID != "sess_1" {
		t.Fatalf("upsert input = %+v calls=%d", deps.presence.upsertInput, deps.presence.upsertCalls)
	}
	if got.OnlineCount != 3 || got.Degraded || got.TTLSeconds != 30 {
		t.Fatalf("result = %+v", got)
	}
}

func TestReaderPresenceRedisFailureReturnsDegradedSuccess(t *testing.T) {
	deps := newCreatePostDeps()
	deps.presence.err = errors.New("redis down")
	service := NewService(deps.asDeps())

	got, err := service.GetReaderPresence(context.Background(), ReaderPresenceQuery{PostID: "post_1"})
	if err != nil {
		t.Fatalf("GetReaderPresence() error = %v", err)
	}
	if got.PostID != "post_1" || got.OnlineCount != 0 || !got.Degraded || got.TTLSeconds != 30 {
		t.Fatalf("result = %+v, want degraded empty presence", got)
	}
}

func TestDeleteReaderSessionIsBestEffort(t *testing.T) {
	deps := newCreatePostDeps()
	deps.presence.err = errors.New("redis down")
	service := NewService(deps.asDeps())

	err := service.DeleteReaderSession(context.Background(), ReaderSessionCommand{
		Actor:     &Actor{UserID: 42},
		PostID:    "post_1",
		SessionID: "sess_1",
	})
	if err != nil {
		t.Fatalf("DeleteReaderSession() error = %v, want best-effort success", err)
	}
	if deps.presence.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", deps.presence.deleteCalls)
	}
}
