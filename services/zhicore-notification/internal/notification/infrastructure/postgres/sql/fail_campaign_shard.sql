UPDATE notification_campaign_shard
SET status = 'FAILED',
    last_error_code = $4,
    next_retry_at = $5 + ($6 * INTERVAL '1 second'),
    updated_at = $5
WHERE id = $1
  AND status = 'PROCESSING'
  AND claimed_by = $2
  AND claim_deadline_at = $3;
