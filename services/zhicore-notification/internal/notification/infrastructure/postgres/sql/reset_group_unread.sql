UPDATE notification_group_state
SET unread_count = 0, updated_at = $2
WHERE recipient_id = $1
