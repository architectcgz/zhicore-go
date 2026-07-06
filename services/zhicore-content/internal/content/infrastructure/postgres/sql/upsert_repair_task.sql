
INSERT INTO content_body_repair_tasks (
    post_id,
    body_id,
    task_type,
    expected_hash,
    observed_hash,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $6)
ON CONFLICT (post_id, task_type, body_id) DO UPDATE
SET expected_hash = EXCLUDED.expected_hash,
    observed_hash = EXCLUDED.observed_hash,
    updated_at = EXCLUDED.updated_at