package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

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

func execTaskStatusUpdate(ctx context.Context, execer sqlExecutor, statement string, args ...any) error {
	result, err := execer.ExecContext(ctx, statement, args...)
	if err != nil {
		return fmt.Errorf("update content body task status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read content body task status update result: %w", err)
	}
	if affected != 1 {
		return ports.ErrTaskClaimLost
	}
	return nil
}

var _ ports.BodyCleanupTaskStore = (*CleanupTaskStore)(nil)
var _ ports.BodyRepairTaskStore = (*RepairTaskStore)(nil)
