UPDATE notification_stats
SET unread_total = GREATEST(unread_total - 1, 0),
    unread_interaction = CASE WHEN $2 = 'INTERACTION' THEN GREATEST(unread_interaction - 1, 0) ELSE unread_interaction END,
    unread_content = CASE WHEN $2 = 'CONTENT' THEN GREATEST(unread_content - 1, 0) ELSE unread_content END,
    unread_social = CASE WHEN $2 = 'SOCIAL' THEN GREATEST(unread_social - 1, 0) ELSE unread_social END,
    unread_system = CASE WHEN $2 = 'SYSTEM' THEN GREATEST(unread_system - 1, 0) ELSE unread_system END,
    unread_security = CASE WHEN $2 = 'SECURITY' THEN GREATEST(unread_security - 1, 0) ELSE unread_security END,
    updated_at = $3
WHERE recipient_id = $1
