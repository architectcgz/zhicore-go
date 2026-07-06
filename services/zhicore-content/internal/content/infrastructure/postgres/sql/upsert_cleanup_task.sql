
INSERT INTO content_body_cleanup_tasks (
    post_id,
    body_id,
    task_type,
    reason,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $5)
ON CONFLICT (body_id, task_type) DO UPDATE
SET post_id = COALESCE(content_body_cleanup_tasks.post_id, EXCLUDED.post_id),
    reason = EXCLUDED.reason,
    updated_at = EXCLUDED.updated_at