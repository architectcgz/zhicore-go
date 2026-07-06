
INSERT INTO %s (
    event_id,
    event_type,
    %s,
    aggregate_type,
    aggregate_id,
    payload_json,
    status,
    occurred_at
)
VALUES ($1, $2, $3, $4, $5, $6, 'PENDING', $7)