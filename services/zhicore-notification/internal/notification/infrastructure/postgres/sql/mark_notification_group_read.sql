UPDATE notifications
SET is_read = TRUE,
    read_at = $3,
    updated_at = $3
WHERE recipient_id = $1
  AND group_id = $2
  AND is_read = FALSE
