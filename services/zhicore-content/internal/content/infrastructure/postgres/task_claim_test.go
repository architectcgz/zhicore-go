package postgres

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestCleanupTaskClaimUsesSkipLockedAndMarksProcessing(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewCleanupTaskStore(NewStore(db, StoreConfig{}))
	now := time.Date(2026, 7, 5, 13, 0, 0, 0, time.UTC)
	staleBefore := now.Add(-5 * time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta(claimCleanupTasksSQL)).
		WithArgs(now, staleBefore, 2, "worker-a").
		WillReturnRows(sqlmock.NewRows([]string{"id", "post_id", "body_id", "task_type", "reason", "attempt_count"}).
			AddRow(int64(1), int64(10), "body_old", "OLD_DRAFT", "draft_replaced", 1).
			AddRow(int64(2), nil, "body_orphan", "ORPHAN_SNAPSHOT", "publish_tx_failed", 3))

	tasks, err := store.Claim(ctx(), ports.TaskClaimRequest{
		WorkerID:    "worker-a",
		Limit:       2,
		Now:         now,
		StaleBefore: staleBefore,
	})
	if err != nil {
		t.Fatalf("Claim(cleanup) error = %v", err)
	}
	if len(tasks) != 2 || tasks[0].ID != 1 || tasks[0].PostID != 10 || tasks[1].PostID != 0 {
		t.Fatalf("tasks = %+v, want two claimed cleanup tasks", tasks)
	}
	if tasks[1].AttemptCount != 3 {
		t.Fatalf("attempt count = %d, want returned incremented count", tasks[1].AttemptCount)
	}
	assertExpectations(t, mock)
}

func TestRepairTaskClaimUsesSkipLockedAndMarksProcessing(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewRepairTaskStore(NewStore(db, StoreConfig{}))
	now := time.Date(2026, 7, 5, 13, 5, 0, 0, time.UTC)
	staleBefore := now.Add(-5 * time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta(claimRepairTasksSQL)).
		WithArgs(now, staleBefore, 1, "worker-b").
		WillReturnRows(sqlmock.NewRows([]string{"id", "post_id", "body_id", "task_type", "expected_hash", "observed_hash", "attempt_count"}).
			AddRow(int64(8), int64(10), "body_pub", "body_hash_mismatch", "sha256:expected", "sha256:observed", 2))

	tasks, err := store.Claim(ctx(), ports.TaskClaimRequest{
		WorkerID:    "worker-b",
		Limit:       1,
		Now:         now,
		StaleBefore: staleBefore,
	})
	if err != nil {
		t.Fatalf("Claim(repair) error = %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != 8 || tasks[0].ExpectedHash != "sha256:expected" || tasks[0].ObservedHash != "sha256:observed" {
		t.Fatalf("tasks = %+v, want repair task metadata", tasks)
	}
	assertExpectations(t, mock)
}

func TestCleanupTaskMarkSucceededRequiresCurrentClaim(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewCleanupTaskStore(NewStore(db, StoreConfig{}))
	completedAt := time.Date(2026, 7, 5, 13, 10, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta(markCleanupTaskSucceededSQL)).
		WithArgs(int64(1), "worker-a", completedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.MarkSucceeded(ctx(), 1, "worker-a", completedAt); err != nil {
		t.Fatalf("MarkSucceeded(cleanup) error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestRepairTaskMarkSucceededDetectsLostClaim(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewRepairTaskStore(NewStore(db, StoreConfig{}))
	resolvedAt := time.Date(2026, 7, 5, 13, 15, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta(markRepairTaskSucceededSQL)).
		WithArgs(int64(8), "stale-worker", resolvedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := store.MarkSucceeded(ctx(), 8, "stale-worker", resolvedAt)
	if !errors.Is(err, ports.ErrTaskClaimLost) {
		t.Fatalf("MarkSucceeded(repair) error = %v, want ErrTaskClaimLost", err)
	}
	assertExpectations(t, mock)
}

func TestCleanupTaskMarkFailedSchedulesRetryOrDeadLetter(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewCleanupTaskStore(NewStore(db, StoreConfig{}))
	now := time.Date(2026, 7, 5, 13, 20, 0, 0, time.UTC)
	nextRetryAt := now.Add(time.Minute)

	mock.ExpectExec(regexp.QuoteMeta(markCleanupTaskFailedSQL)).
		WithArgs(int64(1), "worker-a", 3, "mongo delete failed", nextRetryAt, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(markCleanupTaskFailedSQL)).
		WithArgs(int64(2), "worker-a", 3, "still failing", nextRetryAt, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.MarkFailed(ctx(), ports.TaskFailure{
		TaskID:        1,
		WorkerID:      "worker-a",
		Error:         "mongo delete failed",
		NextRetryAt:   nextRetryAt,
		DeadThreshold: 3,
		Now:           now,
	}); err != nil {
		t.Fatalf("MarkFailed(cleanup retry) error = %v", err)
	}
	if err := store.MarkFailed(ctx(), ports.TaskFailure{
		TaskID:        2,
		WorkerID:      "worker-a",
		Error:         "still failing",
		NextRetryAt:   nextRetryAt,
		DeadThreshold: 3,
		Now:           now,
	}); err != nil {
		t.Fatalf("MarkFailed(cleanup dead threshold) error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestRepairTaskMarkFailedDetectsLostClaim(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewRepairTaskStore(NewStore(db, StoreConfig{}))
	now := time.Date(2026, 7, 5, 13, 25, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta(markRepairTaskFailedSQL)).
		WithArgs(int64(8), "stale-worker", 2, "manual repair needed", now.Add(time.Minute), now).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := store.MarkFailed(ctx(), ports.TaskFailure{
		TaskID:        8,
		WorkerID:      "stale-worker",
		Error:         "manual repair needed",
		NextRetryAt:   now.Add(time.Minute),
		DeadThreshold: 2,
		Now:           now,
	})
	if !errors.Is(err, ports.ErrTaskClaimLost) {
		t.Fatalf("MarkFailed(repair) error = %v, want ErrTaskClaimLost", err)
	}
	assertExpectations(t, mock)
}

func TestTaskClaimSQLPreventsDuplicateMultiInstanceClaims(t *testing.T) {
	for name, sqlText := range map[string]string{
		"cleanup": claimCleanupTasksSQL,
		"repair":  claimRepairTasksSQL,
	} {
		if !regexp.MustCompile(`FOR UPDATE SKIP LOCKED`).MatchString(sqlText) {
			t.Fatalf("%s claim SQL must use FOR UPDATE SKIP LOCKED", name)
		}
		if !regexp.MustCompile(`status = 'PROCESSING'`).MatchString(sqlText) || !regexp.MustCompile(`claimed_by =`).MatchString(sqlText) {
			t.Fatalf("%s claim SQL must atomically mark claimed rows processing", name)
		}
	}
}

func ctx() context.Context {
	return context.Background()
}
