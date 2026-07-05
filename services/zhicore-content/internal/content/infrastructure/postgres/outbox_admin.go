package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type OutboxAdminRepository struct {
	db *sql.DB
}

func NewOutboxAdminRepository(db *sql.DB) *OutboxAdminRepository {
	return &OutboxAdminRepository{db: db}
}

func (r *OutboxAdminRepository) ListOutboxEvents(ctx context.Context, query ports.OutboxEventQuery) (ports.OutboxEventPage, error) {
	offset := (query.Page - 1) * query.Size
	rows, err := r.db.QueryContext(ctx, listAdminOutboxEventsSQL, query.Status, query.EventType, query.Size, offset)
	if err != nil {
		return ports.OutboxEventPage{}, fmt.Errorf("list admin outbox events: %w", err)
	}
	defer rows.Close()

	page := ports.OutboxEventPage{Page: query.Page, Size: query.Size}
	for rows.Next() {
		var item ports.OutboxEventRecord
		var total int64
		if err := rows.Scan(
			&item.EventID,
			&item.EventType,
			&item.AggregateType,
			&item.AggregateID,
			&item.AggregateVersion,
			&item.Status,
			&item.AttemptCount,
			&item.LastError,
			&item.OccurredAt,
			&item.CreatedAt,
			&item.UpdatedAt,
			&total,
		); err != nil {
			return ports.OutboxEventPage{}, fmt.Errorf("scan admin outbox event: %w", err)
		}
		page.Items = append(page.Items, item)
		page.Total = total
	}
	if err := rows.Err(); err != nil {
		return ports.OutboxEventPage{}, fmt.Errorf("iterate admin outbox events: %w", err)
	}
	return page, nil
}

func (r *OutboxAdminRepository) RetryOutboxEvent(ctx context.Context, command ports.OutboxRetryCommand) (ports.OutboxRetryResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ports.OutboxRetryResult{}, fmt.Errorf("begin outbox retry tx: %w", err)
	}
	defer tx.Rollback()

	var result ports.OutboxRetryResult
	if err := tx.QueryRowContext(ctx, retryAdminOutboxEventSQL,
		command.EventID,
		command.AdminUserID,
		command.Reason,
		command.RetriedAt,
	).Scan(&result.EventID, &result.Status, &result.RetryCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ports.ErrOutboxEventNotFound) {
			return ports.OutboxRetryResult{}, ports.ErrOutboxEventNotFound
		}
		return ports.OutboxRetryResult{}, fmt.Errorf("retry admin outbox event: %w", err)
	}
	result.RetriedAt = command.RetriedAt
	if err := tx.Commit(); err != nil {
		return ports.OutboxRetryResult{}, fmt.Errorf("commit outbox retry tx: %w", err)
	}
	return result, nil
}

const listAdminOutboxEventsSQL = `
SELECT
    event_id,
    event_type,
    aggregate_type,
    aggregate_id,
    COALESCE(aggregate_version, 0) AS aggregate_version,
    status,
    attempt_count,
    COALESCE(last_error, '') AS last_error,
    occurred_at,
    created_at,
    updated_at,
    COUNT(*) OVER() AS total_count
FROM outbox_events
WHERE status = $1
  AND ($2 = '' OR event_type = $2)
ORDER BY updated_at DESC, id DESC
LIMIT $3 OFFSET $4`

const retryAdminOutboxEventSQL = `
WITH picked AS (
    SELECT id, event_id, status, attempt_count
    FROM outbox_events
    WHERE event_id = $1
      AND status IN ('FAILED', 'DEAD')
    FOR UPDATE
),
updated AS (
    UPDATE outbox_events AS e
    SET status = 'PENDING',
        claimed_by = NULL,
        claim_started_at = NULL,
        next_retry_at = $4,
        last_error = NULL,
        updated_at = $4
    FROM picked
    WHERE e.id = picked.id
    RETURNING e.event_id, e.status, e.attempt_count, picked.status AS previous_status
),
audit AS (
    INSERT INTO outbox_retry_audit (
        event_id,
        admin_user_id,
        retry_reason,
        previous_status,
        retry_count,
        retried_at,
        created_at
    )
    SELECT event_id, $2, $3, previous_status, attempt_count, $4, $4
    FROM updated
)
SELECT event_id, status, attempt_count
FROM updated`

var _ ports.OutboxAdminRepository = (*OutboxAdminRepository)(nil)
