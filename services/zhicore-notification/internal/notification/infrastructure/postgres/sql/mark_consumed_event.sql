UPDATE consumed_events
SET status = 'CONSUMED', consumed_at = $2, updated_at = $2
WHERE event_id = $1
