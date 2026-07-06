SELECT notification_type, category, target_type, target_id, total_count, unread_count, latest_time, latest_content, latest_actor_ids, aggregated_content
FROM notification_group_state
WHERE recipient_id = $1
  AND ($2 = '' OR category = $2)
  AND ($3 = FALSE OR unread_count > 0)
ORDER BY latest_time DESC, group_key DESC
LIMIT $4
