WITH rebuilt AS (
    INSERT INTO notification_group_state (
        recipient_id,
        group_key,
        notification_type,
        category,
        target_type,
        target_id,
        latest_notification_id,
        total_count,
        unread_count,
        latest_time,
        latest_content,
        latest_actor_ids,
        aggregated_content,
        repair_required,
        created_at,
        updated_at
    )
    SELECT recipient_id,
           group_key,
           notification_type,
           category,
           target_type,
           target_id,
           (ARRAY_AGG(id ORDER BY created_at DESC, id DESC))[1] AS latest_notification_id,
           COUNT(*) AS total_count,
           COUNT(*) FILTER (WHERE is_read = FALSE) AS unread_count,
           MAX(created_at) AS latest_time,
           (ARRAY_AGG(content ORDER BY created_at DESC, id DESC))[1] AS latest_content,
           ARRAY_REMOVE((ARRAY_AGG(actor_id ORDER BY created_at DESC, id DESC))[1:5], NULL) AS latest_actor_ids,
           '{}'::jsonb AS aggregated_content,
           FALSE AS repair_required,
           $2 AS created_at,
           $2 AS updated_at
    FROM notifications
    WHERE recipient_id = $1
    GROUP BY recipient_id, group_key, notification_type, category, target_type, target_id
    RETURNING group_key
)
SELECT COUNT(*) FROM rebuilt;
