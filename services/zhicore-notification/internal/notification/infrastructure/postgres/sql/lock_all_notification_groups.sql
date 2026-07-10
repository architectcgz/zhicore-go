SELECT group_key
FROM notification_group_state
WHERE recipient_id = $1
ORDER BY group_key
FOR UPDATE
