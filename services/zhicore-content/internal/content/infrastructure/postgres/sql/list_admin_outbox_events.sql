
SELECT
    event_id,
    event_type,
    aggregate_type,
    aggregate_id,
    COALESCE(aggregate_version, 0) AS aggregate_version,
    status,
    attempt_count,
    COALESCE(last_error, '') AS last_error,
    occurred_at,
    created_at,
    updated_at,
    COUNT(*) OVER() AS total_count
FROM outbox_events
WHERE status = $1
  AND ($2 = '' OR event_type = $2)
ORDER BY updated_at DESC, id DESC
LIMIT $3 OFFSET $4