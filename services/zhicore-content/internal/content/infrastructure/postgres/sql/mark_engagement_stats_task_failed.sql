UPDATE domain_event_tasks
SET status = CASE WHEN attempt_count >= $3 THEN 'DEAD' ELSE 'FAILED' END,
    last_error = $4,
    next_retry_at = CASE WHEN attempt_count >= $3 THEN NULL ELSE $5 END,
    claimed_by = NULL,
    claimed_at = NULL,
    updated_at = $6
WHERE id = $1
  AND claimed_by = $2
  AND status = 'PROCESSING'
  AND event_type = 'content.engagement.stats_delta'
