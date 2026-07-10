SELECT group_key
FROM notification_group_state
WHERE recipient_id = $1
  AND group_key = $2
FOR UPDATE
