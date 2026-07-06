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

func TestPublishPost(t *testing.T) {
	t.Run("rejects non owner", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.posts.getResult.OwnerID = 2002
		service := NewService(deps.asDeps())

		_, err := service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("error = %v, want ErrForbidden", err)
		}
		if deps.bodies.readCalls != 0 || deps.bodies.writeSnapshotCalls != 0 {
			t.Fatalf("body read/write snapshot = %d/%d, want none", deps.bodies.readCalls, deps.bodies.writeSnapshotCalls)
		}
	})

	t.Run("rejects empty title and missing body", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.posts.getResult.DraftTitle = " "
		service := NewService(deps.asDeps())

		_, err := service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, domain.ErrTitleRequired) {
			t.Fatalf("error = %v, want ErrTitleRequired", err)
		}

		deps = newPublishPostDeps()
		deps.posts.getResult.DraftBodyID = ""
		service = NewService(deps.asDeps())
		_, err = service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, domain.ErrBodyRequired) {
			t.Fatalf("error = %v, want ErrBodyRequired", err)
		}
	})

	t.Run("rejects repeated publish and draft conflict", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.posts.getResult.Status = domain.PostStatusPublished
		service := NewService(deps.asDeps())
		_, err := service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, domain.ErrPostAlreadyPublished) {
			t.Fatalf("error = %v, want ErrPostAlreadyPublished", err)
		}

		deps = newPublishPostDeps()
		deps.posts.getResult.DraftBodyHash = "sha256:server"
		service = NewService(deps.asDeps())
		_, err = service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, domain.ErrDraftConflict) {
			t.Fatalf("error = %v, want ErrDraftConflict", err)
		}
	})

	t.Run("rejects draft body miss and hash conflict", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.bodies.readErr = domain.ErrBodyUnavailable
		service := NewService(deps.asDeps())
		_, err := service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, domain.ErrBodyUnavailable) {
			t.Fatalf("error = %v, want ErrBodyUnavailable", err)
		}
		if deps.repair.appendOutsideCalls != 1 || deps.repair.outsideTasks[0].TaskType != "draft_body_missing" {
			t.Fatalf("repair tasks = %+v, want draft_body_missing", deps.repair.outsideTasks)
		}

		deps = newPublishPostDeps()
		deps.bodies.readResult.ContentHash = "sha256:actual"
		service = NewService(deps.asDeps())
		_, err = service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, domain.ErrBodyInconsistent) {
			t.Fatalf("error = %v, want ErrBodyInconsistent", err)
		}
		if deps.repair.appendOutsideCalls != 1 || deps.repair.outsideTasks[0].TaskType != "body_hash_mismatch" {
			t.Fatalf("repair tasks = %+v, want body_hash_mismatch", deps.repair.outsideTasks)
		}
		if got := deps.repair.outsideTasks[0]; got.ExpectedHash != "sha256:draft" || got.ObservedHash != "sha256:actual" {
			t.Fatalf("repair task = %+v, want expected/observed hash", got)
		}
	})

	t.Run("rejects recomputed body hash mismatch before snapshot write", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.parser.normalized.ContentHash = "sha256:recomputed"
		service := NewService(deps.asDeps())

		_, err := service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, domain.ErrBodyInconsistent) {
			t.Fatalf("error = %v, want ErrBodyInconsistent", err)
		}
		if deps.bodies.writeSnapshotCalls != 0 || deps.posts.publishCalls != 0 {
			t.Fatalf("snapshot/publish calls = %d/%d, want none", deps.bodies.writeSnapshotCalls, deps.posts.publishCalls)
		}
	})

	t.Run("rejects scheduled post through normal publish path", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.posts.getResult.Status = domain.PostStatusScheduled
		service := NewService(deps.asDeps())

		_, err := service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, domain.ErrDraftConflict) {
			t.Fatalf("error = %v, want ErrDraftConflict", err)
		}
		if deps.bodies.writeSnapshotCalls != 0 || deps.posts.publishCalls != 0 {
			t.Fatalf("snapshot/publish calls = %d/%d, want none", deps.bodies.writeSnapshotCalls, deps.posts.publishCalls)
		}
	})

	t.Run("rejects insufficient text and snapshot write failure", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.bodies.readResult.PlainText = "123456789"
		deps.parser.normalized.PlainText = "123456789"
		service := NewService(deps.asDeps())
		_, err := service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, domain.ErrBodyTooShort) {
			t.Fatalf("error = %v, want ErrBodyTooShort", err)
		}

		deps = newPublishPostDeps()
		deps.bodies.writeSnapshotErr = errors.New("mongo write failed")
		service = NewService(deps.asDeps())
		_, err = service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("error = %v, want ErrDependencyUnavailable", err)
		}
		if deps.posts.publishCalls != 0 {
			t.Fatalf("publish calls = %d, want none", deps.posts.publishCalls)
		}
	})

	t.Run("records orphan snapshot cleanup without deleting when PostgreSQL publish fails", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.posts.publishErr = errors.New("pg failed")
		service := NewService(deps.asDeps())

		_, err := service.PublishPost(context.Background(), publishCommand())
		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("error = %v, want ErrDependencyUnavailable", err)
		}
		if deps.bodies.deleteCalls != 0 {
			t.Fatalf("delete calls = %d, want no unsafe direct delete", deps.bodies.deleteCalls)
		}
		if deps.cleanup.appendOutsideCalls != 1 || deps.cleanup.outsideTasks[0].TaskType != "ORPHAN_SNAPSHOT" {
			t.Fatalf("outside cleanup = %+v, want orphan snapshot", deps.cleanup.outsideTasks)
		}
	})

	t.Run("surfaces orphan snapshot cleanup registration failure", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.posts.publishErr = errors.New("pg failed")
		deps.cleanup.err = errors.New("cleanup store down")
		service := NewService(deps.asDeps())

		_, err := service.PublishPost(context.Background(), publishCommand())

		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("error = %v, want ErrDependencyUnavailable", err)
		}
		if !errors.Is(err, deps.cleanup.err) {
			t.Fatalf("error = %v, want wrapped cleanup error", err)
		}
	})

	t.Run("revalidates stored body and cover before publish", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.posts.getResult.DraftCoverFileID = "cover_1"
		deps.parser.normalized = ports.NormalizedBody{
			PlainText:     "publishable body text",
			CanonicalJSON: []byte(`{"schemaVersion":1,"blocks":[]}`),
			ContentHash:   "sha256:draft",
			SizeBytes:     100,
			MediaRefs:     []ports.MediaRef{{FileID: "file_1"}},
		}
		service := NewService(deps.asDeps())

		if _, err := service.PublishPost(context.Background(), publishCommand()); err != nil {
			t.Fatalf("PublishPost returned error: %v", err)
		}
		if deps.parser.calls != 1 {
			t.Fatalf("parser calls = %d, want stored body revalidation", deps.parser.calls)
		}
		if deps.files.validateMediaCalls != 1 || deps.files.mediaRefs[0].FileID != "file_1" {
			t.Fatalf("media validation = %d/%+v, want file_1", deps.files.validateMediaCalls, deps.files.mediaRefs)
		}
		if deps.files.validateCoverCalls != 1 || deps.files.coverFileID != "cover_1" {
			t.Fatalf("cover validation = %d/%s, want cover_1", deps.files.validateCoverCalls, deps.files.coverFileID)
		}
	})

	t.Run("returns media reference error before snapshot write", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.parser.normalized.MediaRefs = []ports.MediaRef{{FileID: "file_missing"}}
		deps.files.err = ports.ErrMediaRefInvalid
		service := NewService(deps.asDeps())

		_, err := service.PublishPost(context.Background(), publishCommand())

		if !errors.Is(err, ErrMediaRefInvalid) {
			t.Fatalf("error = %v, want ErrMediaRefInvalid", err)
		}
		if deps.bodies.writeSnapshotCalls != 0 || deps.posts.publishCalls != 0 {
			t.Fatalf("snapshot/publish calls = %d/%d, want none", deps.bodies.writeSnapshotCalls, deps.posts.publishCalls)
		}
	})

	t.Run("returns cover unavailable before snapshot write", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.posts.getResult.DraftCoverFileID = "cover_missing"
		deps.files.err = ports.ErrCoverUnavailable
		service := NewService(deps.asDeps())

		_, err := service.PublishPost(context.Background(), publishCommand())

		if !errors.Is(err, ErrCoverUnavailable) {
			t.Fatalf("error = %v, want ErrCoverUnavailable", err)
		}
		if deps.bodies.writeSnapshotCalls != 0 || deps.posts.publishCalls != 0 {
			t.Fatalf("snapshot/publish calls = %d/%d, want none", deps.bodies.writeSnapshotCalls, deps.posts.publishCalls)
		}
	})

	t.Run("returns dependency unavailable for file service outage", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.parser.normalized.MediaRefs = []ports.MediaRef{{FileID: "file_1"}}
		deps.files.err = ports.ErrDependencyUnavailable
		service := NewService(deps.asDeps())

		_, err := service.PublishPost(context.Background(), publishCommand())

		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("error = %v, want ErrDependencyUnavailable", err)
		}
		if deps.bodies.writeSnapshotCalls != 0 || deps.posts.publishCalls != 0 {
			t.Fatalf("snapshot/publish calls = %d/%d, want none", deps.bodies.writeSnapshotCalls, deps.posts.publishCalls)
		}
	})

	t.Run("allows exactly ten effective runes", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.bodies.readResult.PlainText = "1234567890"
		deps.parser.normalized.PlainText = "1234567890"
		service := NewService(deps.asDeps())

		if _, err := service.PublishPost(context.Background(), publishCommand()); err != nil {
			t.Fatalf("PublishPost returned error: %v", err)
		}
	})

	t.Run("publishes snapshot and appends complete outbox contract", func(t *testing.T) {
		deps := newPublishPostDeps()
		deps.posts.getResult.DraftSummary = "published summary"
		deps.posts.getResult.DraftCoverFileID = "cover_1"
		service := NewService(deps.asDeps())

		got, err := service.PublishPost(context.Background(), publishCommand())
		if err != nil {
			t.Fatalf("PublishPost returned error: %v", err)
		}
		if got.PostID != "post_1" || got.PostVersion != 6 {
			t.Fatalf("result = %+v, want post_1 version 6", got)
		}
		if deps.posts.publishInput.NewPublishedBodyID != "snapshot_1" {
			t.Fatalf("publish input = %+v, want snapshot body", deps.posts.publishInput)
		}
		if deps.outbox.appendCalls != 1 {
			t.Fatalf("outbox append calls = %d, want 1", deps.outbox.appendCalls)
		}
		gotEvent := deps.outbox.events[0]
		if gotEvent.EventType != "content.post.published" {
			t.Fatalf("outbox = %+v, want content.post.published", deps.outbox.events)
		}
		if gotEvent.PayloadVersion != 1 || gotEvent.AggregateType != "post" {
			t.Fatalf("outbox event = %+v, want payload version 1 aggregate post", gotEvent)
		}
		if gotEvent.AggregateID != "post_1" || gotEvent.AggregateVersion != 6 {
			t.Fatalf("aggregate id/version = %s/%d, want post_1/6", gotEvent.AggregateID, gotEvent.AggregateVersion)
		}
		if !gotEvent.OccurredAt.Equal(deps.clock.now) {
			t.Fatalf("occurredAt = %s, want %s", gotEvent.OccurredAt, deps.clock.now)
		}
		var payload contentevents.PostPublishedPayload
		if err := json.Unmarshal(gotEvent.PayloadJSON, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.PublicID != "post_1" || payload.InternalID != 10 || payload.AuthorID != 1001 {
			t.Fatalf("payload ids = %+v, want public/internal/author ids", payload)
		}
		if payload.Title != "Ready" || payload.Summary != "published summary" || payload.CoverFileID != "cover_1" {
			t.Fatalf("payload content = %+v, want title/summary/cover", payload)
		}
		if !payload.PublishedAt.Equal(deps.clock.now) {
			t.Fatalf("payload publishedAt = %s, want %s", payload.PublishedAt, deps.clock.now)
		}
		if payload.PublishedBodyID != "snapshot_1" || payload.PublishedBodyHash != "sha256:draft" {
			t.Fatalf("payload body = %+v, want snapshot_1/sha256:draft", payload)
		}
		if deps.cleanup.appendCalls != 1 || deps.cleanup.tasks[0].BodyID != "body_draft" {
			t.Fatalf("cleanup tasks = %+v, want draft cleanup", deps.cleanup.tasks)
		}
	})
}

func newPublishPostDeps() createPostDeps {
	deps := newCreatePostDeps()
	deps.posts.getResult = ports.PostRecord{
		ID:                   10,
		PublicID:             "post_1",
		OwnerID:              1001,
		Status:               domain.PostStatusDraft,
		PostVersion:          5,
		DraftTitle:           "Ready",
		DraftBodyID:          "body_draft",
		DraftBodyHash:        "sha256:draft",
		DraftPlainTextLength: 12,
	}
	deps.posts.publishResult = ports.PostRecord{PublicID: "post_1", PostVersion: 6}
	deps.bodies.readResult = ports.StoredBody{
		ID:            "body_draft",
		SchemaVersion: 1,
		Blocks:        ports.Blocks{},
		PlainText:     "publishable body text",
		ContentHash:   "sha256:draft",
		SizeBytes:     100,
	}
	deps.parser.normalized = ports.NormalizedBody{
		PlainText:     "publishable body text",
		CanonicalJSON: []byte(`{"schemaVersion":1,"blocks":[]}`),
		ContentHash:   "sha256:draft",
		SizeBytes:     100,
		BlockCount:    1,
	}
	deps.bodies.snapshotResult = ports.StoredBody{ID: "snapshot_1"}
	return deps
}

func publishCommand() PublishPostCommand {
	return PublishPostCommand{
		Actor:           &Actor{UserID: 1001},
		PostID:          "post_1",
		BasePostVersion: 5,
		DraftBodyID:     "body_draft",
		DraftBodyHash:   "sha256:draft",
	}
}
