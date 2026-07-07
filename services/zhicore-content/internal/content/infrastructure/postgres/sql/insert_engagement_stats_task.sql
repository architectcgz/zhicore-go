INSERT INTO domain_event_tasks (
    task_id,
    event_type,
    aggregate_type,
    aggregate_id,
    payload_json,
    occurred_at,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, $6, $6)
