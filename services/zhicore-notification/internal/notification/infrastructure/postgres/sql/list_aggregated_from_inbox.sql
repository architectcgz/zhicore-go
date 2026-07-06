SELECT notification_type,
       category,
       target_type,
       target_id,
       COUNT(*) AS total_count,
       COUNT(*) FILTER (WHERE is_read = FALSE) AS unread_count,
       MAX(created_at) AS latest_time,
       (ARRAY_AGG(content ORDER BY created_at DESC, id DESC))[1] AS latest_content,
       ARRAY_REMOVE((ARRAY_AGG(actor_id ORDER BY created_at DESC, id DESC))[1:5], NULL) AS latest_actor_ids,
       '{}'::jsonb AS aggregated_content
FROM notifications
WHERE recipient_id = $1
  AND ($2 = '' OR category = $2)
  AND ($3 = FALSE OR is_read = FALSE)
GROUP BY group_key, notification_type, category, target_type, target_id
ORDER BY latest_time DESC, group_key DESC
LIMIT $4
