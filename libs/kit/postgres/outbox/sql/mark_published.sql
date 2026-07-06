
UPDATE %s
SET status = 'PUBLISHED',
    claimed_by = NULL,
    claim_started_at = NULL,
    next_retry_at = NULL,
    last_error = NULL,
    published_at = $1,
    updated_at = $1
WHERE id = $2
  AND status = 'CLAIMING'
  AND claimed_by = $3