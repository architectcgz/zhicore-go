UPDATE notifications
SET is_read = TRUE, read_at = $2, updated_at = $2
WHERE recipient_id = $1 AND is_read = FALSE
