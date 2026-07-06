UPDATE notifications
SET is_read = TRUE, read_at = $3, updated_at = $3
WHERE id = $1 AND recipient_id = $2
