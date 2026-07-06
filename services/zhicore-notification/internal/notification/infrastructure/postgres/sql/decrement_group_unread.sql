UPDATE notification_group_state
SET unread_count = GREATEST(unread_count - 1, 0), updated_at = $3
WHERE recipient_id = $1 AND group_key = $2
