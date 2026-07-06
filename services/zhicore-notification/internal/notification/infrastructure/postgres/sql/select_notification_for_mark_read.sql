SELECT public_id, group_key, is_read, read_at
FROM notifications
WHERE id = $1 AND recipient_id = $2
FOR UPDATE
