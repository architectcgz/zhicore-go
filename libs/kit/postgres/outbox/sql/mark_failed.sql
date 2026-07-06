
UPDATE %s
SET status = $1,
    claimed_by = NULL,
    claim_started_at = NULL,
    attempt_count = $2,
    next_retry_at = $3,
    last_error = $4,
    updated_at = $5
WHERE id = $6
  AND status = 'CLAIMING'
  AND claimed_by = $7