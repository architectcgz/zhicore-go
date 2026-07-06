
UPDATE content_body_repair_tasks
SET status = 'DONE',
    resolved_at = $3,
    updated_at = $3
WHERE id = $1
  AND claimed_by = $2
  AND status = 'PROCESSING'