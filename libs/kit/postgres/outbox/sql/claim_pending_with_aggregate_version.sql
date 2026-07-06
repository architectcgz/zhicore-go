
WITH picked AS (
    SELECT id
    FROM %s
    WHERE (
        status IN ('PENDING', 'FAILED')
        AND (next_retry_at IS NULL OR next_retry_at <= $1)
    )
    -- CLAIMING rows are reclaimed only after their dispatcher lease is stale;
    -- otherwise a crash after claim commit could strand events forever.
    OR (
        status = 'CLAIMING'
        AND claim_started_at < $2
    )
    ORDER BY id
    FOR UPDATE SKIP LOCKED
    LIMIT $4
)
UPDATE %s AS e
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
    e.%s,
    e.aggregate_type,
    e.aggregate_id,
    e.%s,
    e.payload_json,
    e.occurred_at,
    e.attempt_count