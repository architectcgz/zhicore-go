package application

import (
	"context"
	"errors"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestSaveDraftBody(t *testing.T) {
	t.Run("rate limit fail closed before loading post or writing body", func(t *testing.T) {
		deps := newSaveDraftDeps()
		limiter := &recordingRateLimiter{decision: ports.RateLimitDecision{
			Outcome: ports.RateLimitOutcomeDegradedDenyUnavailable,
			Reason:  "redis_unavailable_fail_closed",
		}}
		serviceDeps := deps.asDeps()
		serviceDeps.Limiter = limiter
		service := NewService(serviceDeps)

		_, err := service.SaveDraftBody(context.Background(), saveDraftCommand())

		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("SaveDraftBody() error = %v, want ErrDependencyUnavailable", err)
		}
		if deps.posts.getCalls != 0 || deps.bodies.writeDraftCalls != 0 || deps.tx.calls != 0 {
			t.Fatalf("side effects get/write/tx = %d/%d/%d, want none", deps.posts.getCalls, deps.bodies.writeDraftCalls, deps.tx.calls)
		}
		if len(limiter.requests) != 1 {
			t.Fatalf("rate limit requests = %+v, want one", limiter.requests)
		}
		got := limiter.requests[0]
		if got.LimitType != ports.RateLimitTypeDraftWrite ||
			got.Subject != "actor:1001" ||
			got.Resource != "post_1" ||
			got.Operation != "save_draft_body" {
			t.Fatalf("rate limit request = %+v, want draft write actor/post/operation", got)
		}
	})

	t.Run("rejects non owner before body write", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.posts.getResult.OwnerID = 2002
		service := NewService(deps.asDeps())

		_, err := service.SaveDraftBody(context.Background(), saveDraftCommand())
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("error = %v, want ErrForbidden", err)
		}
		if deps.bodies.writeDraftCalls != 0 || deps.posts.saveCalls != 0 {
			t.Fatalf("body/save calls = %d/%d, want none", deps.bodies.writeDraftCalls, deps.posts.saveCalls)
		}
	})

	t.Run("rejects stale post version", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.posts.getResult.PostVersion = 6
		service := NewService(deps.asDeps())

		_, err := service.SaveDraftBody(context.Background(), saveDraftCommand())
		if !errors.Is(err, domain.ErrDraftConflict) {
			t.Fatalf("error = %v, want ErrDraftConflict", err)
		}
		if deps.bodies.writeDraftCalls != 0 {
			t.Fatalf("body writes = %d, want none", deps.bodies.writeDraftCalls)
		}
	})

	t.Run("rejects stale draft pointer", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.posts.getResult.DraftBodyHash = "sha256:server"
		service := NewService(deps.asDeps())

		_, err := service.SaveDraftBody(context.Background(), saveDraftCommand())
		if !errors.Is(err, domain.ErrDraftConflict) {
			t.Fatalf("error = %v, want ErrDraftConflict", err)
		}
		if deps.bodies.writeDraftCalls != 0 {
			t.Fatalf("body writes = %d, want none", deps.bodies.writeDraftCalls)
		}
	})

	t.Run("rejects scheduled post before parsing or body write", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.posts.getResult.Status = domain.PostStatusScheduled
		service := NewService(deps.asDeps())

		_, err := service.SaveDraftBody(context.Background(), saveDraftCommand())
		if !errors.Is(err, domain.ErrDraftConflict) {
			t.Fatalf("error = %v, want ErrDraftConflict", err)
		}
		if deps.parser.calls != 0 || deps.bodies.writeDraftCalls != 0 || deps.posts.saveCalls != 0 {
			t.Fatalf("parse/write/save calls = %d/%d/%d, want none", deps.parser.calls, deps.bodies.writeDraftCalls, deps.posts.saveCalls)
		}
	})

	t.Run("returns no-op when content hash is unchanged", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.parser.normalized.ContentHash = "sha256:old"
		service := NewService(deps.asDeps())

		got, err := service.SaveDraftBody(context.Background(), saveDraftCommand())
		if err != nil {
			t.Fatalf("SaveDraftBody returned error: %v", err)
		}
		if got.DraftBodyID != "body_old" || got.PostVersion != 5 {
			t.Fatalf("result = %+v, want existing draft pointer", got)
		}
		if deps.bodies.writeDraftCalls != 0 || deps.posts.saveCalls != 0 || deps.cleanup.appendCalls != 0 {
			t.Fatalf("write/save/cleanup calls = %d/%d/%d, want none", deps.bodies.writeDraftCalls, deps.posts.saveCalls, deps.cleanup.appendCalls)
		}
	})

	t.Run("copy-on-write saves new draft and schedules old draft cleanup", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.bodies.draftResult = ports.StoredBody{ID: "body_new"}
		deps.posts.saveResult = ports.PostRecord{PublicID: "post_1", PostVersion: 6, DraftBodyID: "body_new", DraftBodyHash: "sha256:new"}
		service := NewService(deps.asDeps())

		got, err := service.SaveDraftBody(context.Background(), saveDraftCommand())
		if err != nil {
			t.Fatalf("SaveDraftBody returned error: %v", err)
		}
		if got.DraftBodyID != "body_new" || got.DraftBodyHash != "sha256:new" || got.PostVersion != 6 {
			t.Fatalf("result = %+v, want new draft pointer", got)
		}
		if deps.posts.saveInput.NewDraftBodyID != "body_new" || deps.posts.saveInput.NewDraftBodyHash != "sha256:new" {
			t.Fatalf("save input pointer = %+v, want new body", deps.posts.saveInput)
		}
		if deps.cleanup.appendCalls != 1 || deps.cleanup.tasks[0].BodyID != "body_old" {
			t.Fatalf("cleanup tasks = %+v, want old draft cleanup", deps.cleanup.tasks)
		}
	})

	t.Run("records orphan cleanup when PostgreSQL update fails after body write", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.bodies.draftResult = ports.StoredBody{ID: "body_orphan"}
		deps.posts.saveErr = errors.New("pg failed")
		service := NewService(deps.asDeps())

		_, err := service.SaveDraftBody(context.Background(), saveDraftCommand())
		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("error = %v, want ErrDependencyUnavailable", err)
		}
		if deps.cleanup.appendOutsideCalls != 1 {
			t.Fatalf("outside cleanup calls = %d, want 1", deps.cleanup.appendOutsideCalls)
		}
		if got := deps.cleanup.outsideTasks[0]; got.BodyID != "body_orphan" || got.TaskType != "ORPHAN_DRAFT" {
			t.Fatalf("outside cleanup task = %+v, want orphan draft body", got)
		}
	})

	t.Run("returns media reference error before writing new draft", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.parser.normalized.MediaRefs = []ports.MediaRef{{FileID: "file_missing"}}
		deps.files.err = ports.ErrMediaRefInvalid
		service := NewService(deps.asDeps())

		_, err := service.SaveDraftBody(context.Background(), saveDraftCommand())

		if !errors.Is(err, ErrMediaRefInvalid) {
			t.Fatalf("error = %v, want ErrMediaRefInvalid", err)
		}
		if deps.bodies.writeDraftCalls != 0 || deps.posts.saveCalls != 0 {
			t.Fatalf("body/save calls = %d/%d, want none", deps.bodies.writeDraftCalls, deps.posts.saveCalls)
		}
	})

	t.Run("returns dependency unavailable for file service outage", func(t *testing.T) {
		deps := newSaveDraftDeps()
		deps.parser.normalized.MediaRefs = []ports.MediaRef{{FileID: "file_1"}}
		deps.files.err = ports.ErrDependencyUnavailable
		service := NewService(deps.asDeps())

		_, err := service.SaveDraftBody(context.Background(), saveDraftCommand())

		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("error = %v, want ErrDependencyUnavailable", err)
		}
		if deps.bodies.writeDraftCalls != 0 || deps.posts.saveCalls != 0 {
			t.Fatalf("body/save calls = %d/%d, want none", deps.bodies.writeDraftCalls, deps.posts.saveCalls)
		}
	})
}

func newSaveDraftDeps() createPostDeps {
	deps := newCreatePostDeps()
	deps.posts.getResult = ports.PostRecord{
		ID:                   10,
		PublicID:             "post_1",
		OwnerID:              1001,
		Status:               domain.PostStatusDraft,
		PostVersion:          5,
		DraftBodyID:          "body_old",
		DraftBodyHash:        "sha256:old",
		DraftSizeBytes:       100,
		DraftPlainTextLength: 8,
	}
	deps.parser.normalized = ports.NormalizedBody{
		PlainText:     "new body",
		CanonicalJSON: []byte(`{"schemaVersion":1,"blocks":[]}`),
		ContentHash:   "sha256:new",
		SizeBytes:     36,
		BlockCount:    1,
	}
	return deps
}

func saveDraftCommand() SaveDraftBodyCommand {
	return SaveDraftBodyCommand{
		Actor:             &Actor{UserID: 1001},
		PostID:            "post_1",
		BasePostVersion:   5,
		BaseDraftBodyID:   "body_old",
		BaseDraftBodyHash: "sha256:old",
		Body: PostBodyInput{
			SchemaVersion: 1,
			Blocks:        ports.Blocks{},
		},
	}
}
