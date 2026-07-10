SELECT state.group_id,
       state.notification_type,
       state.category,
       state.target_type,
       state.target_id,
       state.total_count,
       state.unread_count,
       state.latest_time,
       state.latest_content,
       state.aggregated_content,
       COALESCE(actors.actor_total_count, 0) AS actor_total_count,
       COALESCE(actors.recent_actors, '[]'::jsonb) AS recent_actors
FROM notification_group_state AS state
LEFT JOIN LATERAL (
    SELECT COUNT(DISTINCT notification.actor_public_id) AS actor_total_count,
           COALESCE((
               SELECT jsonb_agg(
                   jsonb_build_object(
                       'publicId', recent.actor_public_id,
                       'displayName', recent.actor_display_name,
                       'avatarUrl', recent.actor_avatar_url
                   )
                   ORDER BY recent.occurred_at DESC, recent.actor_public_id DESC
               )
               FROM (
                   SELECT latest.actor_public_id, latest.actor_display_name, latest.actor_avatar_url, latest.occurred_at
                   FROM (
                       SELECT DISTINCT ON (item.actor_public_id)
                              item.actor_public_id,
                              item.actor_display_name,
                              item.actor_avatar_url,
                              item.occurred_at,
                              item.id
                       FROM notifications AS item
                       WHERE item.recipient_id = state.recipient_id
                         AND item.group_id = state.group_id
                         AND item.actor_public_id <> ''
                         AND item.actor_display_name <> ''
                       ORDER BY item.actor_public_id, item.occurred_at DESC, item.id DESC
                   ) AS latest
                   ORDER BY latest.occurred_at DESC, latest.actor_public_id DESC
                   LIMIT 3
               ) AS recent
           ), '[]'::jsonb) AS recent_actors
    FROM notifications AS notification
    WHERE notification.recipient_id = state.recipient_id
      AND notification.group_id = state.group_id
      AND notification.actor_public_id <> ''
      AND notification.actor_display_name <> ''
) AS actors ON TRUE
WHERE state.recipient_id = $1
  AND ($2 = '' OR state.category = $2)
  AND ($3 = FALSE OR state.unread_count > 0)
  AND ($4 = '' OR state.latest_time < NULLIF($4, '')::timestamptz OR (state.latest_time = NULLIF($4, '')::timestamptz AND state.group_id < $5))
ORDER BY state.latest_time DESC, state.group_id DESC
LIMIT $6
