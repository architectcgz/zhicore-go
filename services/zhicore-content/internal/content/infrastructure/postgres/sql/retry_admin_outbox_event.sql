
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
FROM updated