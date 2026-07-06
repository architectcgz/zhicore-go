SELECT unread_total,
       unread_interaction,
       unread_content,
       unread_social,
       unread_system,
       unread_security
FROM notification_stats
WHERE recipient_id = $1
