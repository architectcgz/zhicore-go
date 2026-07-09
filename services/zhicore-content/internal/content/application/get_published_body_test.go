package application

import (
	"context"
	"errors"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestGetPublishedPostBody(t *testing.T) {
	t.Run("hides draft and deleted posts", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		deps.posts.bodyPointerResult.Status = domain.PostStatusDraft
		service := NewService(deps.asDeps())

		_, err := service.GetPublishedPostBody(context.Background(), GetPublishedPostBodyQuery{PostID: "post_1"})
		if !errors.Is(err, domain.ErrPostNotFound) {
			t.Fatalf("error = %v, want ErrPostNotFound", err)
		}

		deps = newPublishedBodyDeps()
		deps.posts.bodyPointerResult.Status = domain.PostStatusDeleted
		service = NewService(deps.asDeps())
		_, err = service.GetPublishedPostBody(context.Background(), GetPublishedPostBodyQuery{PostID: "post_1"})
		if !errors.Is(err, domain.ErrPostNotFound) {
			t.Fatalf("error = %v, want ErrPostNotFound", err)
		}
	})

	t.Run("records repair task when body is missing", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		deps.bodies.readErr = domain.ErrBodyUnavailable
		service := NewService(deps.asDeps())

		_, err := service.GetPublishedPostBody(context.Background(), GetPublishedPostBodyQuery{PostID: "post_1"})
		if !errors.Is(err, domain.ErrBodyUnavailable) {
			t.Fatalf("error = %v, want ErrBodyUnavailable", err)
		}
		if deps.repair.appendOutsideCalls != 1 || deps.repair.outsideTasks[0].TaskType != "published_body_missing" {
			t.Fatalf("repair tasks = %+v, want published_body_missing", deps.repair.outsideTasks)
		}
	})

	t.Run("records repair task on hash mismatch", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		deps.bodies.readResult.ContentHash = "sha256:actual"
		service := NewService(deps.asDeps())

		_, err := service.GetPublishedPostBody(context.Background(), GetPublishedPostBodyQuery{PostID: "post_1"})
		if !errors.Is(err, domain.ErrBodyInconsistent) {
			t.Fatalf("error = %v, want ErrBodyInconsistent", err)
		}
		if deps.repair.appendOutsideCalls != 1 || deps.repair.outsideTasks[0].TaskType != "body_hash_mismatch" {
			t.Fatalf("repair tasks = %+v, want body_hash_mismatch", deps.repair.outsideTasks)
		}
	})

	t.Run("records repair task on recomputed hash mismatch", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		deps.parser.normalized.ContentHash = "sha256:recomputed"
		service := NewService(deps.asDeps())

		_, err := service.GetPublishedPostBody(context.Background(), GetPublishedPostBodyQuery{PostID: "post_1"})
		if !errors.Is(err, domain.ErrBodyInconsistent) {
			t.Fatalf("error = %v, want ErrBodyInconsistent", err)
		}
		if deps.repair.appendOutsideCalls != 1 || deps.repair.outsideTasks[0].TaskType != "body_hash_mismatch" {
			t.Fatalf("repair tasks = %+v, want body_hash_mismatch", deps.repair.outsideTasks)
		}
	})

	t.Run("rejects unsupported body schema", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		deps.bodies.readResult.SchemaVersion = 99
		service := NewService(deps.asDeps())

		_, err := service.GetPublishedPostBody(context.Background(), GetPublishedPostBodyQuery{PostID: "post_1"})
		if !errors.Is(err, ErrBodySchemaUnsupported) {
			t.Fatalf("error = %v, want ErrBodySchemaUnsupported", err)
		}
	})

	t.Run("records repair task when stored schema cannot be parsed", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		parseErr := &ports.BodyValidationError{Details: []ports.ValidationDetail{{Path: "blocks", Code: "BODY_SCHEMA_INVALID"}}}
		deps.parser.err = parseErr
		service := NewService(deps.asDeps())

		_, err := service.GetPublishedPostBody(context.Background(), GetPublishedPostBodyQuery{PostID: "post_1"})
		if !errors.Is(err, ErrBodySchemaUnsupported) {
			t.Fatalf("error = %v, want ErrBodySchemaUnsupported", err)
		}
		if deps.repair.appendOutsideCalls != 1 || deps.repair.outsideTasks[0].TaskType != "mongo_read_error_after_pg_published" {
			t.Fatalf("repair tasks = %+v, want schema repair", deps.repair.outsideTasks)
		}
	})

	t.Run("returns published body", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		service := NewService(deps.asDeps())

		got, err := service.GetPublishedPostBody(context.Background(), GetPublishedPostBodyQuery{PostID: "post_1"})
		if err != nil {
			t.Fatalf("GetPublishedPostBody returned error: %v", err)
		}
		if got.BodyID != "body_published" || got.ContentHash != "sha256:published" || got.PlainText != "published body" {
			t.Fatalf("body = %+v, want published body", got)
		}
	})

	t.Run("uses internal caller rate limit and fail closed before reading body", func(t *testing.T) {
		deps := newPublishedBodyDeps()
		limiter := &recordingRateLimiter{decision: ports.RateLimitDecision{
			Outcome: ports.RateLimitOutcomeDegradedDenyUnavailable,
			Reason:  "redis_unavailable_fail_closed",
		}}
		serviceDeps := deps.asDeps()
		serviceDeps.Limiter = limiter
		service := NewService(serviceDeps)

		_, err := service.GetPublishedPostBody(context.Background(), GetPublishedPostBodyQuery{
			PostID:          "post_1",
			CallerService:   "zhicore-search",
			CallerOperation: "search.index_post_body",
		})

		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("GetPublishedPostBody() error = %v, want ErrDependencyUnavailable", err)
		}
		if deps.posts.bodyPointerCalls != 0 || deps.bodies.readCalls != 0 {
			t.Fatalf("body read side effects pointer=%d body=%d, want none", deps.posts.bodyPointerCalls, deps.bodies.readCalls)
		}
		if len(limiter.requests) != 1 {
			t.Fatalf("rate limit requests = %+v, want one", limiter.requests)
		}
		got := limiter.requests[0]
		if got.LimitType != ports.RateLimitTypeInternalClient ||
			got.Subject != "caller:zhicore-search:search.index_post_body" ||
			got.Resource != "post_1" ||
			got.Operation != "get_published_post_body" {
			t.Fatalf("rate limit request = %+v, want internal caller body read", got)
		}
	})
}

func newPublishedBodyDeps() createPostDeps {
	deps := newCreatePostDeps()
	deps.posts.bodyPointerResult = ports.PublishedBodyPointer{
		PostID:                10,
		PublicID:              "post_1",
		Status:                domain.PostStatusPublished,
		PublishedBodyID:       "body_published",
		PublishedBodyHash:     "sha256:published",
		PublishedPlainTextLen: 14,
	}
	deps.bodies.readResult = ports.StoredBody{
		ID:            "body_published",
		SchemaVersion: 1,
		Blocks:        ports.Blocks{},
		CanonicalJSON: []byte(`{"schemaVersion":1,"blocks":[]}`),
		PlainText:     "published body",
		ContentHash:   "sha256:published",
		SizeBytes:     36,
	}
	deps.parser.normalized = ports.NormalizedBody{
		PlainText:     "published body",
		CanonicalJSON: []byte(`{"schemaVersion":1,"blocks":[]}`),
		ContentHash:   "sha256:published",
		SizeBytes:     36,
		BlockCount:    1,
	}
	return deps
}
