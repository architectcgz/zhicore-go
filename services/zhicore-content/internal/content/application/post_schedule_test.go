package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestPostScheduleCommands(t *testing.T) {
	t.Run("schedules draft after validating body and cover", func(t *testing.T) {
		deps := newSchedulePostDeps()
		deps.posts.getResult.DraftCoverFileID = "cover_1"
		deps.parser.normalized.MediaRefs = []ports.MediaRef{{FileID: "file_1"}}
		service := NewService(deps.asDeps())

		got, err := service.SchedulePost(context.Background(), scheduleCommand(deps.clock.now.Add(time.Hour)))

		if err != nil {
			t.Fatalf("SchedulePost() error = %v", err)
		}
		if got.PostID != "post_1" || got.PostVersion != 6 || got.Status != string(domain.PostStatusScheduled) ||
			!got.ScheduledAt.Equal(deps.clock.now.Add(time.Hour)) {
			t.Fatalf("result = %+v, want scheduled post_1 version 6", got)
		}
		if deps.posts.scheduleCalls != 1 || deps.posts.scheduleInput.DraftBodyID != "body_draft" {
			t.Fatalf("schedule calls/input = %d/%+v", deps.posts.scheduleCalls, deps.posts.scheduleInput)
		}
		if deps.files.validateMediaCalls != 1 || deps.files.validateCoverCalls != 1 {
			t.Fatalf("file validation media/cover = %d/%d, want 1/1", deps.files.validateMediaCalls, deps.files.validateCoverCalls)
		}
	})

	t.Run("rejects invalid schedule state and past time before side effects", func(t *testing.T) {
		deps := newSchedulePostDeps()
		service := NewService(deps.asDeps())
		_, err := service.SchedulePost(context.Background(), scheduleCommand(deps.clock.now))
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("time error = %v, want ErrInvalidArgument", err)
		}
		if deps.bodies.readCalls != 0 || deps.posts.scheduleCalls != 0 {
			t.Fatalf("body/schedule calls = %d/%d, want none", deps.bodies.readCalls, deps.posts.scheduleCalls)
		}

		deps = newSchedulePostDeps()
		deps.posts.getResult.Status = domain.PostStatusPublished
		service = NewService(deps.asDeps())
		_, err = service.SchedulePost(context.Background(), scheduleCommand(deps.clock.now.Add(time.Hour)))
		if !errors.Is(err, domain.ErrDraftConflict) {
			t.Fatalf("status error = %v, want ErrDraftConflict", err)
		}
	})

	t.Run("rejects schedule body conflicts and validation errors", func(t *testing.T) {
		deps := newSchedulePostDeps()
		deps.posts.getResult.DraftBodyHash = "sha256:server"
		service := NewService(deps.asDeps())
		_, err := service.SchedulePost(context.Background(), scheduleCommand(deps.clock.now.Add(time.Hour)))
		if !errors.Is(err, domain.ErrDraftConflict) {
			t.Fatalf("conflict error = %v, want ErrDraftConflict", err)
		}

		deps = newSchedulePostDeps()
		deps.parser.normalized.PlainText = "short"
		service = NewService(deps.asDeps())
		_, err = service.SchedulePost(context.Background(), scheduleCommand(deps.clock.now.Add(time.Hour)))
		if !errors.Is(err, domain.ErrBodyTooShort) {
			t.Fatalf("body error = %v, want ErrBodyTooShort", err)
		}
	})

	t.Run("cancels scheduled post back to draft", func(t *testing.T) {
		deps := newSchedulePostDeps()
		deps.posts.getResult.Status = domain.PostStatusScheduled
		deps.posts.cancelScheduleResult = ports.PostRecord{PublicID: "post_1", Status: domain.PostStatusDraft, PostVersion: 7}
		service := NewService(deps.asDeps())

		got, err := service.CancelSchedule(context.Background(), PostLifecycleCommand{
			Actor:           &Actor{UserID: 1001},
			PostID:          "post_1",
			BasePostVersion: 6,
		})

		if err != nil {
			t.Fatalf("CancelSchedule() error = %v", err)
		}
		if got.PostID != "post_1" || got.PostVersion != 7 || got.Status != string(domain.PostStatusDraft) {
			t.Fatalf("result = %+v, want draft version 7", got)
		}
		if deps.posts.cancelScheduleCalls != 1 || deps.posts.cancelScheduleInput.BasePostVersion != 6 {
			t.Fatalf("cancel calls/input = %d/%+v", deps.posts.cancelScheduleCalls, deps.posts.cancelScheduleInput)
		}
	})

	t.Run("rejects cancel when post is not scheduled", func(t *testing.T) {
		deps := newSchedulePostDeps()
		deps.posts.getResult.Status = domain.PostStatusDraft
		service := NewService(deps.asDeps())

		_, err := service.CancelSchedule(context.Background(), lifecycleCommand())

		if !errors.Is(err, domain.ErrPostNotPublished) {
			t.Fatalf("error = %v, want ErrPostNotPublished", err)
		}
		if deps.posts.cancelScheduleCalls != 0 {
			t.Fatalf("cancel calls = %d, want none", deps.posts.cancelScheduleCalls)
		}
	})
}

func newSchedulePostDeps() createPostDeps {
	deps := newPostLifecycleDeps(domain.PostStatusDraft)
	deps.posts.scheduleResult = ports.PostRecord{PublicID: "post_1", Status: domain.PostStatusScheduled, PostVersion: 6}
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
	return deps
}

func scheduleCommand(scheduledAt time.Time) SchedulePostCommand {
	return SchedulePostCommand{
		Actor:           &Actor{UserID: 1001},
		PostID:          "post_1",
		BasePostVersion: 5,
		DraftBodyID:     "body_draft",
		DraftBodyHash:   "sha256:draft",
		ScheduledAt:     scheduledAt,
	}
}
