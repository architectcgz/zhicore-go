UPDATE notification_campaign_shard
SET status = 'FAILED',
    last_error_code = $2,
    next_retry_at = $3 + ($4 * INTERVAL '1 second'),
    updated_at = $3
WHERE id = $1;
