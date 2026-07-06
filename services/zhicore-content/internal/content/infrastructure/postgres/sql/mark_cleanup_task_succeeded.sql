
UPDATE content_body_cleanup_tasks
SET status = 'DONE',
    completed_at = $3,
    updated_at = $3
WHERE id = $1
  AND claimed_by = $2
  AND status = 'PROCESSING'