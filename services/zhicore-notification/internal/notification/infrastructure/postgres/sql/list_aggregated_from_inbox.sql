WITH groups AS (
    SELECT group_id,
           notification_type,
           category,
           target_type,
           target_id,
           COUNT(*) AS total_count,
           COUNT(*) FILTER (WHERE is_read = FALSE) AS unread_count,
           MAX(occurred_at) AS latest_time,
           (ARRAY_AGG(content ORDER BY occurred_at DESC, id DESC))[1] AS latest_content,
           (ARRAY_AGG(payload ORDER BY occurred_at DESC, id DESC))[1] AS aggregated_content
    FROM notifications
    WHERE recipient_id = $1
      AND ($2 = '' OR category = $2)
      AND ($3 = FALSE OR is_read = FALSE)
    GROUP BY group_id, group_key, notification_type, category, target_type, target_id
), page AS (
    SELECT *
    FROM groups
    WHERE ($4 = '' OR latest_time < NULLIF($4, '')::timestamptz OR (latest_time = NULLIF($4, '')::timestamptz AND group_id < $5))
    ORDER BY latest_time DESC, group_id DESC
    LIMIT $6
)
SELECT page.group_id,
       page.notification_type,
       page.category,
       page.target_type,
       page.target_id,
       page.total_count,
       page.unread_count,
       page.latest_time,
       page.latest_content,
       page.aggregated_content,
       COALESCE(actors.actor_total_count, 0) AS actor_total_count,
       COALESCE(actors.recent_actors, '[]'::jsonb) AS recent_actors
FROM page
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
                       WHERE item.recipient_id = $1
                         AND item.group_id = page.group_id
                         AND item.actor_public_id <> ''
                         AND item.actor_display_name <> ''
                       ORDER BY item.actor_public_id, item.occurred_at DESC, item.id DESC
                   ) AS latest
                   ORDER BY latest.occurred_at DESC, latest.actor_public_id DESC
                   LIMIT 3
               ) AS recent
           ), '[]'::jsonb) AS recent_actors
    FROM notifications AS notification
    WHERE notification.recipient_id = $1
      AND notification.group_id = page.group_id
      AND notification.actor_public_id <> ''
      AND notification.actor_display_name <> ''
) AS actors ON TRUE
ORDER BY page.latest_time DESC, page.group_id DESC
