SELECT unread_total
FROM notification_stats
WHERE recipient_id = $1
