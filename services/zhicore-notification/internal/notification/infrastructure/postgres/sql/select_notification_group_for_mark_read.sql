SELECT group_key, category, unread_count
FROM notification_group_state
WHERE recipient_id = $1
  AND group_id = $2
FOR UPDATE
