
INSERT INTO outbox_events (
    event_id,
    event_type,
    payload_version,
    aggregate_type,
    aggregate_id,
    aggregate_version,
    payload_json,
    occurred_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)