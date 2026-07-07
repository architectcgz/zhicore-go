UPDATE notification_campaign_shard
SET processed_count = processed_count + $4,
    success_count = success_count + $5,
    skipped_count = skipped_count + $6,
    failed_count = failed_count + $7,
    follower_cursor = CASE WHEN $9 THEN $8 ELSE follower_cursor END,
    next_follower_cursor = $8,
    status = CASE WHEN $9 THEN 'PENDING' ELSE 'COMPLETED' END,
    claimed_by = '',
    claimed_at = NULL,
    claim_deadline_at = NULL,
    next_retry_at = NULL,
    updated_at = $10
WHERE id = $1
  AND status = 'PROCESSING'
  AND claimed_by = $2
  AND claim_deadline_at = $3;
