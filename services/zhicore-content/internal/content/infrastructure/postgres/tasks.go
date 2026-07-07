package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const engagementStatsDeltaEventType = "content.engagement.stats_delta"

type CleanupTaskStore struct {
	store *Store
}

func NewCleanupTaskStore(store *Store) *CleanupTaskStore {
	return &CleanupTaskStore{store: store}
}

func (s *CleanupTaskStore) Append(ctx context.Context, tx ports.Tx, task ports.BodyCleanupTask) error {
	execer, err := s.store.execer(tx)
	if err != nil {
		return err
	}
	return appendCleanupTask(ctx, execer, task)
}

func (s *CleanupTaskStore) AppendOutsideTx(ctx context.Context, task ports.BodyCleanupTask) error {
	return appendCleanupTask(ctx, s.store.db, task)
}

func (s *CleanupTaskStore) Claim(ctx context.Context, request ports.TaskClaimRequest) ([]ports.BodyCleanupTaskClaim, error) {
	rows, err := s.store.db.QueryContext(ctx, claimCleanupTasksSQL, request.Now, request.StaleBefore, request.Limit, request.WorkerID)
	if err != nil {
		return nil, fmt.Errorf("claim content body cleanup tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]ports.BodyCleanupTaskClaim, 0)
	for rows.Next() {
		var task ports.BodyCleanupTaskClaim
		var postID sql.NullInt64
		if err := rows.Scan(&task.ID, &postID, &task.BodyID, &task.TaskType, &task.Reason, &task.AttemptCount); err != nil {
			return nil, fmt.Errorf("scan content body cleanup task claim: %w", err)
		}
		if postID.Valid {
			task.PostID = postID.Int64
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content body cleanup task claims: %w", err)
	}
	return tasks, nil
}

func (s *CleanupTaskStore) MarkSucceeded(ctx context.Context, taskID int64, workerID string, completedAt time.Time) error {
	return execTaskStatusUpdate(ctx, s.store.db, markCleanupTaskSucceededSQL, taskID, workerID, completedAt)
}

func (s *CleanupTaskStore) MarkFailed(ctx context.Context, failure ports.TaskFailure) error {
	return execTaskStatusUpdate(
		ctx,
		s.store.db,
		markCleanupTaskFailedSQL,
		failure.TaskID,
		failure.WorkerID,
		failure.DeadThreshold,
		failure.Error,
		failure.NextRetryAt,
		failure.Now,
	)
}

type RepairTaskStore struct {
	store *Store
}

func NewRepairTaskStore(store *Store) *RepairTaskStore {
	return &RepairTaskStore{store: store}
}

func (s *RepairTaskStore) Append(ctx context.Context, tx ports.Tx, task ports.BodyRepairTask) error {
	execer, err := s.store.execer(tx)
	if err != nil {
		return err
	}
	return appendRepairTask(ctx, execer, task)
}

func (s *RepairTaskStore) AppendOutsideTx(ctx context.Context, task ports.BodyRepairTask) error {
	return appendRepairTask(ctx, s.store.db, task)
}

func (s *RepairTaskStore) Claim(ctx context.Context, request ports.TaskClaimRequest) ([]ports.BodyRepairTaskClaim, error) {
	rows, err := s.store.db.QueryContext(ctx, claimRepairTasksSQL, request.Now, request.StaleBefore, request.Limit, request.WorkerID)
	if err != nil {
		return nil, fmt.Errorf("claim content body repair tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]ports.BodyRepairTaskClaim, 0)
	for rows.Next() {
		var task ports.BodyRepairTaskClaim
		var expectedHash, observedHash sql.NullString
		if err := rows.Scan(&task.ID, &task.PostID, &task.BodyID, &task.TaskType, &expectedHash, &observedHash, &task.AttemptCount); err != nil {
			return nil, fmt.Errorf("scan content body repair task claim: %w", err)
		}
		if expectedHash.Valid {
			task.ExpectedHash = expectedHash.String
		}
		if observedHash.Valid {
			task.ObservedHash = observedHash.String
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content body repair task claims: %w", err)
	}
	return tasks, nil
}

func (s *RepairTaskStore) MarkSucceeded(ctx context.Context, taskID int64, workerID string, resolvedAt time.Time) error {
	return execTaskStatusUpdate(ctx, s.store.db, markRepairTaskSucceededSQL, taskID, workerID, resolvedAt)
}

func (s *RepairTaskStore) MarkFailed(ctx context.Context, failure ports.TaskFailure) error {
	return execTaskStatusUpdate(
		ctx,
		s.store.db,
		markRepairTaskFailedSQL,
		failure.TaskID,
		failure.WorkerID,
		failure.DeadThreshold,
		failure.Error,
		failure.NextRetryAt,
		failure.Now,
	)
}

type EngagementStatsTaskStore struct {
	store *Store
}

func NewEngagementStatsTaskStore(store *Store) *EngagementStatsTaskStore {
	return &EngagementStatsTaskStore{store: store}
}

func (s *EngagementStatsTaskStore) Append(ctx context.Context, tx ports.Tx, task ports.EngagementStatsDeltaTask) error {
	execer, err := s.store.execer(tx)
	if err != nil {
		return err
	}
	if task.TaskID == "" {
		taskID, err := s.store.eventIDs.NewID()
		if err != nil {
			return fmt.Errorf("generate content engagement stats task id: %w", err)
		}
		task.TaskID = taskID
	}
	payload, err := json.Marshal(engagementStatsTaskPayload{
		PostInternalID: task.PostInternalID,
		PostID:         task.PostID,
		Metric:         task.Metric,
		Delta:          task.Delta,
	})
	if err != nil {
		return fmt.Errorf("marshal content engagement stats task payload: %w", err)
	}
	if _, err := execer.ExecContext(ctx, insertEngagementStatsTaskSQL,
		task.TaskID,
		engagementStatsDeltaEventType,
		"post",
		task.PostID,
		payload,
		task.OccurredAt,
	); err != nil {
		return fmt.Errorf("insert content engagement stats task: %w", err)
	}
	return nil
}

func (s *EngagementStatsTaskStore) Claim(ctx context.Context, request ports.TaskClaimRequest) ([]ports.EngagementStatsDeltaClaim, error) {
	rows, err := s.store.db.QueryContext(ctx, claimEngagementStatsTasksSQL, request.Now, request.StaleBefore, request.Limit, request.WorkerID)
	if err != nil {
		return nil, fmt.Errorf("claim content engagement stats tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]ports.EngagementStatsDeltaClaim, 0)
	for rows.Next() {
		var task ports.EngagementStatsDeltaClaim
		if err := rows.Scan(&task.ID, &task.TaskID, &task.PostInternalID, &task.PostID, &task.Metric, &task.Delta, &task.AttemptCount); err != nil {
			return nil, fmt.Errorf("scan content engagement stats task claim: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content engagement stats task claims: %w", err)
	}
	return tasks, nil
}

func (s *EngagementStatsTaskStore) ApplyClaimed(ctx context.Context, task ports.EngagementStatsDeltaClaim, workerID string, appliedAt time.Time) error {
	tx, err := s.store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin content engagement stats task transaction: %w", err)
	}
	defer tx.Rollback()

	if err := execTaskStatusUpdate(ctx, tx, applyEngagementStatsTaskSQL, task.ID, workerID, task.PostInternalID, task.Metric, task.Delta, appliedAt); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit content engagement stats task transaction: %w", err)
	}
	return nil
}

func (s *EngagementStatsTaskStore) MarkFailed(ctx context.Context, failure ports.TaskFailure) error {
	return execTaskStatusUpdate(
		ctx,
		s.store.db,
		markEngagementStatsTaskFailedSQL,
		failure.TaskID,
		failure.WorkerID,
		failure.DeadThreshold,
		failure.Error,
		failure.NextRetryAt,
		failure.Now,
	)
}

type engagementStatsTaskPayload struct {
	PostInternalID int64  `json:"postInternalId"`
	PostID         string `json:"postId"`
	Metric         string `json:"metric"`
	Delta          int    `json:"delta"`
}

func execTaskStatusUpdate(ctx context.Context, execer sqlExecutor, statement string, args ...any) error {
	result, err := execer.ExecContext(ctx, statement, args...)
	if err != nil {
		return fmt.Errorf("update content task status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read content task status update result: %w", err)
	}
	if affected != 1 {
		return ports.ErrTaskClaimLost
	}
	return nil
}

var _ ports.BodyCleanupTaskStore = (*CleanupTaskStore)(nil)
var _ ports.BodyRepairTaskStore = (*RepairTaskStore)(nil)
var _ ports.EngagementStatsTaskStore = (*EngagementStatsTaskStore)(nil)
