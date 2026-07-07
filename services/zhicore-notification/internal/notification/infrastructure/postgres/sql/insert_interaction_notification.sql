INSERT INTO notifications (
    id, public_id, recipient_id, actor_id, category, notification_type, event_code, importance,
    target_type, target_id, source_event_id, dedupe_key, group_key, title, content, payload,
    occurred_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13, $14, $15, $16,
    $17, $18, $18
)
ON CONFLICT DO NOTHING
RETURNING id
