package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestEngagementStatsWorkerAppliesClaimedDeltas(t *testing.T) {
	deps := newCreatePostDeps()
	deps.engagementStats.claims = []ports.EngagementStatsDeltaClaim{
		{ID: 31, TaskID: "engagement_stats_post_1_like_1", PostInternalID: 10, PostID: "post_1", Metric: "LIKE", Delta: 1, AttemptCount: 1},
		{ID: 32, TaskID: "engagement_stats_post_1_favorite_1", PostInternalID: 10, PostID: "post_1", Metric: "FAVORITE", Delta: -1, AttemptCount: 1},
	}
	worker := newTestEngagementStatsWorker(deps)

	err := worker.RunUntilIdle(context.Background())
	if err != nil {
		t.Fatalf("RunUntilIdle() error = %v", err)
	}
	if deps.engagementStats.claimCalls != 2 {
		t.Fatalf("claim calls = %d, want first batch and final empty claim", deps.engagementStats.claimCalls)
	}
	if len(deps.engagementStats.applied) != 2 {
		t.Fatalf("applied = %+v, want both deltas applied", deps.engagementStats.applied)
	}
	if deps.engagementStats.applied[0].Metric != "LIKE" || deps.engagementStats.applied[0].Delta != 1 {
		t.Fatalf("first applied delta = %+v, want LIKE +1", deps.engagementStats.applied[0])
	}
	if len(deps.engagementStats.failed) != 0 {
		t.Fatalf("failed = %+v, want none", deps.engagementStats.failed)
	}
}

func TestEngagementStatsWorkerSchedulesRetryWhenApplyFails(t *testing.T) {
	deps := newCreatePostDeps()
	deps.engagementStats.claims = []ports.EngagementStatsDeltaClaim{
		{ID: 33, TaskID: "engagement_stats_post_1_like_retry", PostInternalID: 10, PostID: "post_1", Metric: "LIKE", Delta: 1, AttemptCount: 2},
	}
	deps.engagementStats.applyErr = errors.New("postgres unavailable")
	worker := newTestEngagementStatsWorker(deps)

	err := worker.RunUntilIdle(context.Background())
	if err != nil {
		t.Fatalf("RunUntilIdle() error = %v", err)
	}
	if len(deps.engagementStats.applied) != 1 {
		t.Fatalf("applied = %+v, want one attempted apply", deps.engagementStats.applied)
	}
	if len(deps.engagementStats.failed) != 1 {
		t.Fatalf("failed = %+v, want retry failure", deps.engagementStats.failed)
	}
	got := deps.engagementStats.failed[0]
	if got.TaskID != 33 || got.WorkerID != "engagement-stats-worker-a" || got.NextRetryAt != deps.clock.now.Add(time.Minute) || got.DeadThreshold != 5 {
		t.Fatalf("failure = %+v, want retry policy from worker config", got)
	}
}

func TestEngagementStatsWorkerIgnoresLostClaimDuringApply(t *testing.T) {
	deps := newCreatePostDeps()
	deps.engagementStats.claims = []ports.EngagementStatsDeltaClaim{
		{ID: 34, TaskID: "engagement_stats_post_1_like_lost", PostInternalID: 10, PostID: "post_1", Metric: "LIKE", Delta: 1, AttemptCount: 1},
	}
	deps.engagementStats.applyErr = ports.ErrTaskClaimLost
	worker := newTestEngagementStatsWorker(deps)

	err := worker.RunUntilIdle(context.Background())
	if err != nil {
		t.Fatalf("RunUntilIdle() error = %v", err)
	}
	if len(deps.engagementStats.failed) != 0 {
		t.Fatalf("failed = %+v, want no failed mark after lost claim", deps.engagementStats.failed)
	}
}

func newTestEngagementStatsWorker(deps createPostDeps) *EngagementStatsWorker {
	return NewEngagementStatsWorker(EngagementStatsWorkerDeps{
		Tasks: deps.engagementStats,
		Clock: deps.clock,
	}, EngagementStatsWorkerConfig{
		WorkerID:        "engagement-stats-worker-a",
		BatchSize:       10,
		StaleClaimAfter: 5 * time.Minute,
		RetryBackoff:    time.Minute,
		DeadThreshold:   5,
	})
}
