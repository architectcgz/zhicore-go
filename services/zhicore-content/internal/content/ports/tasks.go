package ports

import (
	"context"
	"errors"
	"time"
)

var ErrTaskClaimLost = errors.New("task claim lost")

type BodyCleanupTaskStore interface {
	Append(ctx context.Context, tx Tx, task BodyCleanupTask) error
	AppendOutsideTx(ctx context.Context, task BodyCleanupTask) error
	Claim(ctx context.Context, request TaskClaimRequest) ([]BodyCleanupTaskClaim, error)
	MarkSucceeded(ctx context.Context, taskID int64, workerID string, completedAt time.Time) error
	MarkFailed(ctx context.Context, failure TaskFailure) error
}

type BodyRepairTaskStore interface {
	Append(ctx context.Context, tx Tx, task BodyRepairTask) error
	AppendOutsideTx(ctx context.Context, task BodyRepairTask) error
	Claim(ctx context.Context, request TaskClaimRequest) ([]BodyRepairTaskClaim, error)
	MarkSucceeded(ctx context.Context, taskID int64, workerID string, resolvedAt time.Time) error
	MarkFailed(ctx context.Context, failure TaskFailure) error
}

type EngagementStatsTaskStore interface {
	Append(ctx context.Context, tx Tx, task EngagementStatsDeltaTask) error
	Claim(ctx context.Context, request TaskClaimRequest) ([]EngagementStatsDeltaClaim, error)
	ApplyClaimed(ctx context.Context, task EngagementStatsDeltaClaim, workerID string, appliedAt time.Time) error
	MarkFailed(ctx context.Context, failure TaskFailure) error
}

type BodyCleanupTask struct {
	PostID    int64
	BodyID    string
	TaskType  string
	Reason    string
	CreatedAt time.Time
}

type BodyRepairTask struct {
	PostID       int64
	BodyID       string
	TaskType     string
	ExpectedHash string
	ObservedHash string
	CreatedAt    time.Time
}

type EngagementStatsDeltaTask struct {
	TaskID         string
	PostInternalID int64
	PostID         string
	Metric         string
	Delta          int
	OccurredAt     time.Time
}

type TaskClaimRequest struct {
	WorkerID    string
	Limit       int
	Now         time.Time
	StaleBefore time.Time
}

type TaskFailure struct {
	TaskID        int64
	WorkerID      string
	Error         string
	NextRetryAt   time.Time
	DeadThreshold int
	Now           time.Time
}

type BodyCleanupTaskClaim struct {
	ID           int64
	PostID       int64
	BodyID       string
	TaskType     string
	Reason       string
	AttemptCount int
}

type BodyRepairTaskClaim struct {
	ID           int64
	PostID       int64
	BodyID       string
	TaskType     string
	ExpectedHash string
	ObservedHash string
	AttemptCount int
}

type EngagementStatsDeltaClaim struct {
	ID             int64
	TaskID         string
	PostInternalID int64
	PostID         string
	Metric         string
	Delta          int
	AttemptCount   int
}
