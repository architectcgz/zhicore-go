package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestBodyCleanupWorker(t *testing.T) {
	t.Run("deletes unreferenced body and marks task succeeded", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.cleanup.claimResults = [][]ports.BodyCleanupTaskClaim{{
			{ID: 11, BodyID: "body_old", TaskType: "OLD_DRAFT", AttemptCount: 1},
		}}
		worker := newTestCleanupWorker(deps)

		err := worker.RunUntilIdle(context.Background())
		if err != nil {
			t.Fatalf("RunUntilIdle() error = %v", err)
		}
		if deps.posts.referenceChecks != 1 || deps.posts.referenceBodyID != "body_old" {
			t.Fatalf("reference checks = %d/%q, want body_old", deps.posts.referenceChecks, deps.posts.referenceBodyID)
		}
		if deps.bodies.deleteCalls != 1 || deps.bodies.deleteBodyID != "body_old" {
			t.Fatalf("delete = %d/%q, want exact body delete", deps.bodies.deleteCalls, deps.bodies.deleteBodyID)
		}
		if len(deps.cleanup.succeeded) != 1 || deps.cleanup.succeeded[0].taskID != 11 {
			t.Fatalf("succeeded = %+v, want task 11", deps.cleanup.succeeded)
		}
		if len(deps.cleanup.failed) != 0 {
			t.Fatalf("failed = %+v, want none", deps.cleanup.failed)
		}
	})

	t.Run("does not delete referenced body and schedules retry", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.cleanup.claimResults = [][]ports.BodyCleanupTaskClaim{{
			{ID: 12, BodyID: "body_live", TaskType: "ORPHAN_SNAPSHOT", AttemptCount: 2},
		}}
		deps.posts.bodyReferenced = true
		worker := newTestCleanupWorker(deps)

		err := worker.RunUntilIdle(context.Background())
		if err != nil {
			t.Fatalf("RunUntilIdle() error = %v", err)
		}
		if deps.bodies.deleteCalls != 0 {
			t.Fatalf("delete calls = %d, want none for referenced body", deps.bodies.deleteCalls)
		}
		if len(deps.cleanup.failed) != 1 {
			t.Fatalf("failed = %+v, want one retry", deps.cleanup.failed)
		}
		got := deps.cleanup.failed[0]
		if got.TaskID != 12 || got.NextRetryAt != deps.clock.now.Add(time.Minute) || got.DeadThreshold != 3 {
			t.Fatalf("failure = %+v, want retry with backoff and threshold", got)
		}
		if !strings.Contains(got.Error, "still referenced") {
			t.Fatalf("failure error = %q, want referenced reason", got.Error)
		}
	})

	t.Run("treats missing Mongo body as idempotent success", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.cleanup.claimResults = [][]ports.BodyCleanupTaskClaim{{
			{ID: 13, BodyID: "body_missing", TaskType: "OLD_DRAFT", AttemptCount: 1},
		}}
		worker := newTestCleanupWorker(deps)

		err := worker.RunUntilIdle(context.Background())
		if err != nil {
			t.Fatalf("RunUntilIdle() error = %v", err)
		}
		if deps.bodies.deleteCalls != 1 || len(deps.cleanup.succeeded) != 1 {
			t.Fatalf("delete/succeeded = %d/%+v, want idempotent success", deps.bodies.deleteCalls, deps.cleanup.succeeded)
		}
	})

	t.Run("schedules retry when Mongo delete fails", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.cleanup.claimResults = [][]ports.BodyCleanupTaskClaim{{
			{ID: 14, BodyID: "body_retry", TaskType: "OLD_DRAFT", AttemptCount: 1},
		}}
		deps.bodies.deleteErr = errors.New("mongo delete failed")
		worker := newTestCleanupWorker(deps)

		err := worker.RunUntilIdle(context.Background())
		if err != nil {
			t.Fatalf("RunUntilIdle() error = %v", err)
		}
		if len(deps.cleanup.failed) != 1 || deps.cleanup.failed[0].TaskID != 14 {
			t.Fatalf("failed = %+v, want task 14 retry", deps.cleanup.failed)
		}
		if len(deps.cleanup.succeeded) != 0 {
			t.Fatalf("succeeded = %+v, want none", deps.cleanup.succeeded)
		}
	})

	t.Run("does not claim another batch after context cancellation", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.cleanup.claimResults = [][]ports.BodyCleanupTaskClaim{
			{{ID: 15, BodyID: "body_first", TaskType: "OLD_DRAFT", AttemptCount: 1}},
			{{ID: 16, BodyID: "body_second", TaskType: "OLD_DRAFT", AttemptCount: 1}},
		}
		ctx, cancel := context.WithCancel(context.Background())
		deps.bodies.afterDelete = cancel
		worker := newTestCleanupWorker(deps)

		err := worker.RunUntilIdle(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("RunUntilIdle() error = %v, want context.Canceled", err)
		}
		if deps.cleanup.claimCalls != 1 {
			t.Fatalf("claim calls = %d, want no second claim after cancel", deps.cleanup.claimCalls)
		}
		if deps.bodies.deleteBodyID != "body_first" {
			t.Fatalf("deleted body = %q, want first only", deps.bodies.deleteBodyID)
		}
	})
}

func newTestCleanupWorker(deps createPostDeps) *BodyCleanupWorker {
	return NewBodyCleanupWorker(BodyCleanupWorkerDeps{
		Tasks:      deps.cleanup,
		Bodies:     deps.bodies,
		References: deps.posts,
		Clock:      deps.clock,
	}, BodyCleanupWorkerConfig{
		WorkerID:        "cleanup-worker-a",
		BatchSize:       10,
		StaleClaimAfter: 5 * time.Minute,
		RetryBackoff:    time.Minute,
		DeadThreshold:   3,
	})
}
