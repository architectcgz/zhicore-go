package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	contentevents "github.com/architectcgz/zhicore-go/libs/contracts/events/content"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestPostLifecycleCommands(t *testing.T) {
	t.Run("unpublishes published post and writes visibility event in transaction", func(t *testing.T) {
		deps := newPostLifecycleDeps(domain.PostStatusPublished)
		service := NewService(deps.asDeps())

		got, err := service.UnpublishPost(context.Background(), lifecycleCommand())

		if err != nil {
			t.Fatalf("UnpublishPost() error = %v", err)
		}
		assertLifecycleResult(t, got, domain.PostStatusDraft, deps.clock.now)
		if deps.posts.unpublishCalls != 1 || deps.posts.unpublishInput.BasePostVersion != 5 {
			t.Fatalf("unpublish calls/input = %d/%+v, want version guarded mutation", deps.posts.unpublishCalls, deps.posts.unpublishInput)
		}
		assertVisibilityEvent(t, deps.outbox.events, "post_1", 6, "PUBLIC", "UNPUBLISHED", false, "AUTHOR_UNPUBLISHED")
	})

	t.Run("rejects unpublish for non owner, draft, and deleted posts", func(t *testing.T) {
		deps := newPostLifecycleDeps(domain.PostStatusPublished)
		deps.posts.getResult.OwnerID = 2002
		service := NewService(deps.asDeps())
		_, err := service.UnpublishPost(context.Background(), lifecycleCommand())
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("non owner error = %v, want ErrForbidden", err)
		}
		if deps.posts.unpublishCalls != 0 || deps.outbox.appendCalls != 0 {
			t.Fatalf("mutation/outbox calls = %d/%d, want none", deps.posts.unpublishCalls, deps.outbox.appendCalls)
		}

		deps = newPostLifecycleDeps(domain.PostStatusDraft)
		service = NewService(deps.asDeps())
		_, err = service.UnpublishPost(context.Background(), lifecycleCommand())
		if !errors.Is(err, domain.ErrPostNotPublished) {
			t.Fatalf("draft error = %v, want ErrPostNotPublished", err)
		}

		deps = newPostLifecycleDeps(domain.PostStatusDeleted)
		service = NewService(deps.asDeps())
		_, err = service.UnpublishPost(context.Background(), lifecycleCommand())
		if !errors.Is(err, domain.ErrPostDeleted) {
			t.Fatalf("deleted error = %v, want ErrPostDeleted", err)
		}
	})

	t.Run("deletes post and writes deleted visibility event without body deletion", func(t *testing.T) {
		deps := newPostLifecycleDeps(domain.PostStatusPublished)
		service := NewService(deps.asDeps())

		got, err := service.DeletePost(context.Background(), lifecycleCommand())

		if err != nil {
			t.Fatalf("DeletePost() error = %v", err)
		}
		assertLifecycleResult(t, got, domain.PostStatusDeleted, deps.clock.now)
		if deps.posts.deletePostCalls != 1 || deps.bodies.deleteCalls != 0 {
			t.Fatalf("deletePost/bodyDelete calls = %d/%d, want soft delete only", deps.posts.deletePostCalls, deps.bodies.deleteCalls)
		}
		assertVisibilityEvent(t, deps.outbox.events, "post_1", 6, "PUBLIC", "DELETED", false, "AUTHOR_DELETED")
	})

	t.Run("rejects repeated delete as already deleted", func(t *testing.T) {
		deps := newPostLifecycleDeps(domain.PostStatusDeleted)
		service := NewService(deps.asDeps())

		_, err := service.DeletePost(context.Background(), lifecycleCommand())

		if !errors.Is(err, domain.ErrPostDeleted) {
			t.Fatalf("error = %v, want ErrPostDeleted", err)
		}
		if deps.posts.deletePostCalls != 0 || deps.outbox.appendCalls != 0 {
			t.Fatalf("delete/outbox calls = %d/%d, want none", deps.posts.deletePostCalls, deps.outbox.appendCalls)
		}
	})

	t.Run("restores deleted post to draft and writes visibility event", func(t *testing.T) {
		deps := newPostLifecycleDeps(domain.PostStatusDeleted)
		service := NewService(deps.asDeps())

		got, err := service.RestorePost(context.Background(), lifecycleCommand())

		if err != nil {
			t.Fatalf("RestorePost() error = %v", err)
		}
		assertLifecycleResult(t, got, domain.PostStatusDraft, deps.clock.now)
		if deps.posts.restoreCalls != 1 || deps.posts.restoreInput.BasePostVersion != 5 {
			t.Fatalf("restore calls/input = %d/%+v, want version guarded mutation", deps.posts.restoreCalls, deps.posts.restoreInput)
		}
		assertVisibilityEvent(t, deps.outbox.events, "post_1", 6, "DELETED", "UNPUBLISHED", false, "AUTHOR_RESTORED")
	})

	t.Run("rejects restore for non deleted post", func(t *testing.T) {
		deps := newPostLifecycleDeps(domain.PostStatusPublished)
		service := NewService(deps.asDeps())

		_, err := service.RestorePost(context.Background(), lifecycleCommand())

		if !errors.Is(err, domain.ErrPostNotFound) {
			t.Fatalf("error = %v, want ErrPostNotFound for non deleted restore", err)
		}
		if deps.posts.restoreCalls != 0 || deps.outbox.appendCalls != 0 {
			t.Fatalf("restore/outbox calls = %d/%d, want none", deps.posts.restoreCalls, deps.outbox.appendCalls)
		}
	})

	t.Run("maps repository conflict and dependency failures", func(t *testing.T) {
		deps := newPostLifecycleDeps(domain.PostStatusPublished)
		deps.posts.unpublishErr = domain.ErrDraftConflict
		service := NewService(deps.asDeps())
		_, err := service.UnpublishPost(context.Background(), lifecycleCommand())
		if !errors.Is(err, domain.ErrDraftConflict) {
			t.Fatalf("conflict error = %v, want ErrDraftConflict", err)
		}

		deps = newPostLifecycleDeps(domain.PostStatusPublished)
		deps.posts.unpublishErr = errors.New("pg down")
		service = NewService(deps.asDeps())
		_, err = service.UnpublishPost(context.Background(), lifecycleCommand())
		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("dependency error = %v, want ErrDependencyUnavailable", err)
		}
	})
}

func newPostLifecycleDeps(status domain.PostStatus) createPostDeps {
	deps := newCreatePostDeps()
	deps.posts.getResult = ports.PostRecord{
		ID:                    10,
		PublicID:              "post_1",
		OwnerID:               1001,
		Status:                status,
		PostVersion:           5,
		DraftTitle:            "Draft",
		DraftBodyID:           "body_draft",
		DraftBodyHash:         "sha256:draft",
		PublishedTitle:        "Published",
		PublishedBodyID:       "body_pub",
		PublishedBodyHash:     "sha256:pub",
		PublishedPlainTextLen: 42,
	}
	deps.posts.unpublishResult = ports.PostRecord{PublicID: "post_1", Status: domain.PostStatusDraft, PostVersion: 6}
	deps.posts.deletePostResult = ports.PostRecord{PublicID: "post_1", Status: domain.PostStatusDeleted, PostVersion: 6}
	deps.posts.restoreResult = ports.PostRecord{PublicID: "post_1", Status: domain.PostStatusDraft, PostVersion: 6}
	return deps
}

func lifecycleCommand() PostLifecycleCommand {
	return PostLifecycleCommand{
		Actor:           &Actor{UserID: 1001},
		PostID:          "post_1",
		BasePostVersion: 5,
	}
}

func assertLifecycleResult(t *testing.T, got PostLifecycleResult, wantStatus domain.PostStatus, wantUpdatedAt any) {
	t.Helper()
	if got.PostID != "post_1" || got.PostVersion != 6 || got.Status != string(wantStatus) {
		t.Fatalf("result = %+v, want post_1 version 6 status %s", got, wantStatus)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatalf("updatedAt is zero")
	}
}

func assertVisibilityEvent(t *testing.T, events []ports.OutboxEvent, postID string, version int64, oldVisibility, newVisibility string, publicVisible bool, reason string) {
	t.Helper()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1: %+v", len(events), events)
	}
	event := events[0]
	if event.EventType != "content.post.visibility_changed" || event.AggregateType != "post" {
		t.Fatalf("event = %+v, want content.post.visibility_changed/post", event)
	}
	if event.AggregateID != postID || event.AggregateVersion != version {
		t.Fatalf("aggregate = %s/%d, want %s/%d", event.AggregateID, event.AggregateVersion, postID, version)
	}
	var payload contentevents.PostVisibilityChangedPayload
	if err := json.Unmarshal(event.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal visibility payload: %v", err)
	}
	if payload.PublicID != postID || payload.OldVisibility != oldVisibility || payload.NewVisibility != newVisibility ||
		payload.PublicVisible != publicVisible || payload.Reason != reason {
		t.Fatalf("payload = %+v, want %s->%s visible=%v reason=%s", payload, oldVisibility, newVisibility, publicVisible, reason)
	}
}
