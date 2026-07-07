package postgres

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestEngagementStatsTaskAppendUsesDomainEventTasks(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewEngagementStatsTaskStore(NewStore(db, StoreConfig{}))
	runner := NewTransactionRunner(db)
	now := time.Date(2026, 7, 7, 13, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(insertEngagementStatsTaskSQL)).
		WithArgs(
			"engagement_stats_post_1_like_1",
			"content.engagement.stats_delta",
			"post",
			"post_1",
			sqlmock.AnyArg(),
			now,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		return store.Append(ctx, tx, ports.EngagementStatsDeltaTask{
			TaskID:         "engagement_stats_post_1_like_1",
			PostInternalID: 10,
			PostID:         "post_1",
			Metric:         "LIKE",
			Delta:          1,
			OccurredAt:     now,
		})
	})
	if err != nil {
		t.Fatalf("Append(engagement stats task) error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestEngagementStatsTaskClaimFiltersStatsDeltaEvents(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewEngagementStatsTaskStore(NewStore(db, StoreConfig{}))
	now := time.Date(2026, 7, 7, 13, 5, 0, 0, time.UTC)
	staleBefore := now.Add(-5 * time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta(claimEngagementStatsTasksSQL)).
		WithArgs(now, staleBefore, 2, "worker-a").
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "post_internal_id", "post_id", "metric", "delta", "attempt_count"}).
			AddRow(int64(41), "engagement_stats_post_1_like_1", int64(10), "post_1", "LIKE", 1, 1).
			AddRow(int64(42), "engagement_stats_post_1_favorite_1", int64(10), "post_1", "FAVORITE", -1, 2))

	tasks, err := store.Claim(ctx(), ports.TaskClaimRequest{
		WorkerID:    "worker-a",
		Limit:       2,
		Now:         now,
		StaleBefore: staleBefore,
	})
	if err != nil {
		t.Fatalf("Claim(engagement stats) error = %v", err)
	}
	if len(tasks) != 2 || tasks[0].Metric != "LIKE" || tasks[1].Delta != -1 || tasks[1].AttemptCount != 2 {
		t.Fatalf("tasks = %+v, want decoded LIKE/FAVORITE deltas", tasks)
	}
	assertExpectations(t, mock)
}

func TestEngagementStatsTaskApplyUpdatesPostStatsAndMarksDoneAtomically(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewEngagementStatsTaskStore(NewStore(db, StoreConfig{}))
	appliedAt := time.Date(2026, 7, 7, 13, 10, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(applyEngagementStatsTaskSQL)).
		WithArgs(int64(41), "worker-a", int64(10), "LIKE", 1, appliedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := store.ApplyClaimed(ctx(), ports.EngagementStatsDeltaClaim{
		ID:             41,
		TaskID:         "engagement_stats_post_1_like_1",
		PostInternalID: 10,
		PostID:         "post_1",
		Metric:         "LIKE",
		Delta:          1,
	}, "worker-a", appliedAt)
	if err != nil {
		t.Fatalf("ApplyClaimed() error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestEngagementStatsTaskApplyDetectsLostClaim(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewEngagementStatsTaskStore(NewStore(db, StoreConfig{}))
	appliedAt := time.Date(2026, 7, 7, 13, 15, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(applyEngagementStatsTaskSQL)).
		WithArgs(int64(41), "stale-worker", int64(10), "LIKE", 1, appliedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err := store.ApplyClaimed(ctx(), ports.EngagementStatsDeltaClaim{
		ID:             41,
		PostInternalID: 10,
		Metric:         "LIKE",
		Delta:          1,
	}, "stale-worker", appliedAt)
	if !errors.Is(err, ports.ErrTaskClaimLost) {
		t.Fatalf("ApplyClaimed() error = %v, want ErrTaskClaimLost", err)
	}
	assertExpectations(t, mock)
}

func TestEngagementStatsTaskMarkFailedUsesDomainEventTaskStatus(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewEngagementStatsTaskStore(NewStore(db, StoreConfig{}))
	now := time.Date(2026, 7, 7, 13, 20, 0, 0, time.UTC)
	nextRetryAt := now.Add(time.Minute)

	mock.ExpectExec(regexp.QuoteMeta(markEngagementStatsTaskFailedSQL)).
		WithArgs(int64(41), "worker-a", 5, "postgres unavailable", nextRetryAt, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.MarkFailed(ctx(), ports.TaskFailure{
		TaskID:        41,
		WorkerID:      "worker-a",
		Error:         "postgres unavailable",
		NextRetryAt:   nextRetryAt,
		DeadThreshold: 5,
		Now:           now,
	}); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestEngagementMutationSQLDoesNotLockPostsOrSynchronouslyUpdateStats(t *testing.T) {
	for name, sqlText := range map[string]string{
		"like":       mutateLikeEngagementSQL,
		"unlike":     mutateUnlikeEngagementSQL,
		"favorite":   mutateFavoriteEngagementSQL,
		"unfavorite": mutateUnfavoriteEngagementSQL,
	} {
		upper := strings.ToUpper(sqlText)
		if strings.Contains(upper, "FOR UPDATE") {
			t.Fatalf("%s mutation SQL must not lock posts with FOR UPDATE", name)
		}
		if regexp.MustCompile(`(?i)UPDATE\s+post_stats`).MatchString(sqlText) {
			t.Fatalf("%s mutation SQL must not update post_stats synchronously", name)
		}
		if regexp.MustCompile(`(?i)UPDATE\s+posts`).MatchString(sqlText) || strings.Contains(upper, "POST_VERSION +") {
			t.Fatalf("%s mutation SQL must not bump posts.post_version", name)
		}
	}
}
