SELECT 1
FROM notification_group_state
WHERE recipient_id = $1
  AND group_id = $2
