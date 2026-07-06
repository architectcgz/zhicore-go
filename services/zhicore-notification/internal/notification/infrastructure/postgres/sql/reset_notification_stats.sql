UPDATE notification_stats
SET unread_total = 0,
    unread_interaction = 0,
    unread_content = 0,
    unread_social = 0,
    unread_system = 0,
    unread_security = 0,
    updated_at = $2
WHERE recipient_id = $1
