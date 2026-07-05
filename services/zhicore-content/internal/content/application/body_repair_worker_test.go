package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestBodyRepairWorker(t *testing.T) {
	t.Run("marks consistency incidents for manual repair without mutating bodies", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.repair.claimResults = [][]ports.BodyRepairTaskClaim{{
			{ID: 21, PostID: 10, BodyID: "body_missing", TaskType: "published_body_missing", ExpectedHash: "sha256:expected", AttemptCount: 1},
			{ID: 22, PostID: 10, BodyID: "body_bad_hash", TaskType: "body_hash_mismatch", ExpectedHash: "sha256:expected", ObservedHash: "sha256:observed", AttemptCount: 1},
			{ID: 23, PostID: 10, BodyID: "body_unreadable", TaskType: "mongo_read_error_after_pg_published", ExpectedHash: "sha256:expected", AttemptCount: 1},
		}}
		worker := newTestRepairWorker(deps)

		err := worker.RunUntilIdle(context.Background())
		if err != nil {
			t.Fatalf("RunUntilIdle() error = %v", err)
		}
		if deps.bodies.readCalls != 0 || deps.bodies.writeDraftCalls != 0 || deps.bodies.writeSnapshotCalls != 0 || deps.bodies.deleteCalls != 0 {
			t.Fatalf("body calls read/writeDraft/writeSnapshot/delete = %d/%d/%d/%d, want none",
				deps.bodies.readCalls, deps.bodies.writeDraftCalls, deps.bodies.writeSnapshotCalls, deps.bodies.deleteCalls)
		}
		if len(deps.repair.succeeded) != 0 {
			t.Fatalf("succeeded = %+v, want no fake automatic repair success", deps.repair.succeeded)
		}
		if len(deps.repair.failed) != 3 {
			t.Fatalf("failed = %+v, want three manual repair records", deps.repair.failed)
		}
		if deps.repair.failed[0].DeadThreshold != 1 || deps.repair.failed[0].NextRetryAt != deps.clock.now.Add(time.Minute) {
			t.Fatalf("failure policy = %+v, want dead-letter threshold and backoff", deps.repair.failed[0])
		}
		if !strings.Contains(deps.repair.failed[1].Error, "body_hash_mismatch") ||
			!strings.Contains(deps.repair.failed[1].Error, "sha256:expected") ||
			!strings.Contains(deps.repair.failed[1].Error, "sha256:observed") {
			t.Fatalf("hash mismatch failure error = %q, want task and hashes", deps.repair.failed[1].Error)
		}
	})

	t.Run("keeps invalid repair task in failure path", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.repair.claimResults = [][]ports.BodyRepairTaskClaim{{
			{ID: 24, TaskType: "published_body_missing", AttemptCount: 1},
		}}
		worker := newTestRepairWorker(deps)

		err := worker.RunUntilIdle(context.Background())
		if err != nil {
			t.Fatalf("RunUntilIdle() error = %v", err)
		}
		if len(deps.repair.failed) != 1 || !strings.Contains(deps.repair.failed[0].Error, "body id is empty") {
			t.Fatalf("failed = %+v, want invalid repair task failure", deps.repair.failed)
		}
	})
}

func newTestRepairWorker(deps createPostDeps) *BodyRepairWorker {
	return NewBodyRepairWorker(BodyRepairWorkerDeps{
		Tasks: deps.repair,
		Clock: deps.clock,
	}, BodyRepairWorkerConfig{
		WorkerID:        "repair-worker-a",
		BatchSize:       10,
		StaleClaimAfter: 5 * time.Minute,
		RetryBackoff:    time.Minute,
		DeadThreshold:   1,
	})
}
