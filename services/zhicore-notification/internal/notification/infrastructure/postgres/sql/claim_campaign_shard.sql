UPDATE notification_campaign_shard
SET status = 'PROCESSING',
    claimed_by = $1,
    claimed_at = $2,
    claim_deadline_at = $2 + ($3 * INTERVAL '1 second'),
    attempt_count = attempt_count + 1,
    updated_at = $2
WHERE id = (
    SELECT id
    FROM notification_campaign_shard
    WHERE (
        status IN ('PENDING', 'FAILED')
        AND (next_retry_at IS NULL OR next_retry_at <= $2)
    )
    OR (
        status = 'PROCESSING'
        AND claim_deadline_at < $2
    )
    ORDER BY id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, campaign_id, follower_cursor, attempt_count, claim_deadline_at;
