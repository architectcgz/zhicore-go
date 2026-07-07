INSERT INTO notification_group_state (
    recipient_id, group_key, notification_type, category, target_type, target_id,
    latest_notification_id, total_count, unread_count, latest_time, latest_content,
    latest_actor_ids, aggregated_content, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, 1, 1, $8, $9,
    CASE WHEN $10::BIGINT IS NULL THEN '{}'::BIGINT[] ELSE ARRAY[$10::BIGINT] END,
    $11, $12, $12
)
ON CONFLICT (recipient_id, group_key) DO UPDATE SET
    latest_notification_id = EXCLUDED.latest_notification_id,
    total_count = notification_group_state.total_count + 1,
    unread_count = notification_group_state.unread_count + 1,
    latest_time = EXCLUDED.latest_time,
    latest_content = EXCLUDED.latest_content,
    latest_actor_ids = CASE
        WHEN $10::BIGINT IS NULL THEN notification_group_state.latest_actor_ids
        ELSE (ARRAY_PREPEND($10::BIGINT, ARRAY_REMOVE(notification_group_state.latest_actor_ids, $10::BIGINT)))[1:5]
    END,
    aggregated_content = EXCLUDED.aggregated_content,
    updated_at = EXCLUDED.updated_at
