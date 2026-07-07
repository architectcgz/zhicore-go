WITH picked AS (
    SELECT id
    FROM domain_event_tasks
    WHERE event_type = 'content.engagement.stats_delta'
      AND (
        (
            status IN ('PENDING', 'FAILED')
            AND (next_retry_at IS NULL OR next_retry_at <= $1)
        )
        OR (
            status = 'PROCESSING'
            AND claimed_at < $2
        )
      )
    ORDER BY priority DESC, id
    LIMIT $3
    FOR UPDATE SKIP LOCKED
)
UPDATE domain_event_tasks AS task
SET status = 'PROCESSING',
    claimed_by = $4,
    claimed_at = $1,
    attempt_count = attempt_count + 1,
    updated_at = $1
FROM picked
WHERE task.id = picked.id
RETURNING
    task.id,
    task.task_id,
    (task.payload_json ->> 'postInternalId')::BIGINT AS post_internal_id,
    task.payload_json ->> 'postId' AS post_id,
    task.payload_json ->> 'metric' AS metric,
    (task.payload_json ->> 'delta')::INT AS delta,
    task.attempt_count
