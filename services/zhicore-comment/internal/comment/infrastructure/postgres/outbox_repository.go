package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

var ErrOutboxClaimLost = errors.New("outbox claim lost")

const claimPendingOutboxSQL = `
WITH picked AS (
    SELECT id
    FROM outbox_events
	    WHERE (
	        status IN ('PENDING', 'FAILED')
	        AND (next_retry_at IS NULL OR next_retry_at <= $1)
	    )
	    -- CLAIMING rows are only reclaimed after the dispatcher lease is stale.
	    -- The original owner may have crashed after committing the claim but
	    -- before publishing or marking the event, so leaving stale claims out
	    -- would permanently strand those outbox events.
	    OR (
	        status = 'CLAIMING'
	        AND claim_started_at < $2
    )
    ORDER BY id
    FOR UPDATE SKIP LOCKED
    LIMIT $4
)
UPDATE outbox_events AS e
SET status = 'CLAIMING',
    claimed_by = $3,
    claim_started_at = $1,
    updated_at = $1
FROM picked
WHERE e.id = picked.id
RETURNING
    e.id,
    e.event_id,
    e.event_type,
    e.payload_version,
    e.aggregate_type,
    e.aggregate_id,
    e.payload_json,
    e.occurred_at,
    e.attempt_count`

const markOutboxPublishedSQL = `
UPDATE outbox_events
SET status = 'PUBLISHED',
    claimed_by = NULL,
    claim_started_at = NULL,
    next_retry_at = NULL,
    last_error = NULL,
    published_at = $1,
    updated_at = $1
WHERE id = $2
  AND status = 'CLAIMING'
  AND claimed_by = $3`

const markOutboxFailedSQL = `
UPDATE outbox_events
SET status = $1,
    claimed_by = NULL,
    claim_started_at = NULL,
    attempt_count = $2,
    next_retry_at = $3,
    last_error = $4,
    updated_at = $5
WHERE id = $6
  AND status = 'CLAIMING'
  AND claimed_by = $7`

type OutboxDispatchRepository struct {
	db *sql.DB
}

func NewOutboxDispatchRepository(db *sql.DB) *OutboxDispatchRepository {
	return &OutboxDispatchRepository{db: db}
}

func (r *OutboxDispatchRepository) ClaimPendingOutbox(ctx context.Context, options ports.OutboxClaimOptions) ([]ports.OutboxEvent, error) {
	staleBefore := options.Now.Add(-options.StaleAfter)
	rows, err := r.db.QueryContext(ctx, claimPendingOutboxSQL, options.Now, staleBefore, options.DispatcherID, options.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("claim comment outbox events: %w", err)
	}
	defer rows.Close()

	var events []ports.OutboxEvent
	for rows.Next() {
		var event ports.OutboxEvent
		if err := rows.Scan(
			&event.ID,
			&event.EventID,
			&event.EventType,
			&event.PayloadVersion,
			&event.AggregateType,
			&event.AggregateID,
			&event.Payload,
			&event.OccurredAt,
			&event.AttemptCount,
		); err != nil {
			return nil, fmt.Errorf("scan claimed comment outbox event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed comment outbox events: %w", err)
	}
	return events, nil
}

func (r *OutboxDispatchRepository) MarkOutboxPublished(ctx context.Context, published ports.OutboxPublished) error {
	result, err := r.db.ExecContext(ctx, markOutboxPublishedSQL, published.PublishedAt, published.ID, published.DispatcherID)
	if err != nil {
		return fmt.Errorf("mark comment outbox published: %w", err)
	}
	return requireClaimedRow(result)
}

func (r *OutboxDispatchRepository) MarkOutboxFailed(ctx context.Context, failure ports.OutboxFailure) error {
	status := "FAILED"
	var nextRetryAt any = failure.NextRetryAt
	if failure.Dead {
		status = "DEAD"
		nextRetryAt = nil
	}
	result, err := r.db.ExecContext(ctx, markOutboxFailedSQL,
		status,
		failure.AttemptCount,
		nextRetryAt,
		failure.LastError,
		failure.FailedAt,
		failure.ID,
		failure.DispatcherID,
	)
	if err != nil {
		return fmt.Errorf("mark comment outbox failed: %w", err)
	}
	return requireClaimedRow(result)
}

func requireClaimedRow(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read comment outbox affected rows: %w", err)
	}
	if affected == 0 {
		// A zero-row conditional update means another dispatcher finished or
		// reclaimed the event; callers must not blindly overwrite that state.
		return ErrOutboxClaimLost
	}
	return nil
}
