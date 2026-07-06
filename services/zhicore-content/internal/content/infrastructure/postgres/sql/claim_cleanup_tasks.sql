
WITH picked AS (
    SELECT id
    FROM content_body_cleanup_tasks
    WHERE (
        status IN ('PENDING', 'FAILED')
        AND (next_retry_at IS NULL OR next_retry_at <= $1)
    )
    OR (
        status = 'PROCESSING'
        AND claimed_at < $2
    )
    ORDER BY id
    LIMIT $3
    FOR UPDATE SKIP LOCKED
)
UPDATE content_body_cleanup_tasks AS task
SET status = 'PROCESSING',
    claimed_by = $4,
    claimed_at = $1,
    attempt_count = attempt_count + 1,
    updated_at = $1
FROM picked
WHERE task.id = picked.id
RETURNING task.id, task.post_id, task.body_id, task.task_type, task.reason, task.attempt_count