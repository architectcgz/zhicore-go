SELECT category, COUNT(*)
FROM notifications
WHERE recipient_id = $1 AND is_read = FALSE
GROUP BY category
