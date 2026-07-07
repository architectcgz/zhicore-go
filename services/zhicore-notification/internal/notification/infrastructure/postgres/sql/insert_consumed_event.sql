INSERT INTO consumed_events (event_id, event_type, routing_key, consumer_name, payload_hash, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (event_id) DO NOTHING
RETURNING event_id
