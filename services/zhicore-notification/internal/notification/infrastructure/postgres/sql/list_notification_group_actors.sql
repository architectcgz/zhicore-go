WITH actor_events AS (
    SELECT actor_public_id,
           COUNT(*) AS event_count,
           MAX(occurred_at) AS latest_occurred_at
    FROM notifications
    WHERE recipient_id = $1
      AND group_id = $2
      AND actor_public_id <> ''
      AND actor_display_name <> ''
    GROUP BY actor_public_id
)
SELECT actor_events.actor_public_id,
       snapshot.actor_display_name,
       snapshot.actor_avatar_url,
       actor_events.event_count,
       actor_events.latest_occurred_at
FROM actor_events
CROSS JOIN LATERAL (
    SELECT actor_display_name, actor_avatar_url
    FROM notifications
    WHERE recipient_id = $1
      AND group_id = $2
      AND actor_public_id = actor_events.actor_public_id
    ORDER BY occurred_at DESC, id DESC
    LIMIT 1
) AS snapshot
WHERE ($3 = '' OR actor_events.latest_occurred_at < NULLIF($3, '')::timestamptz OR (actor_events.latest_occurred_at = NULLIF($3, '')::timestamptz AND actor_events.actor_public_id < $4))
ORDER BY actor_events.latest_occurred_at DESC, actor_events.actor_public_id DESC
LIMIT $5
