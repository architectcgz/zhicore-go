SELECT COUNT(*)
FROM notifications
WHERE recipient_id = $1 AND is_read = FALSE
