UPDATE notification_stats
SET unread_total = GREATEST(0, unread_total - $3),
    unread_interaction = GREATEST(0, unread_interaction - CASE WHEN $2 = 'INTERACTION' THEN $3 ELSE 0 END),
    unread_content = GREATEST(0, unread_content - CASE WHEN $2 = 'CONTENT' THEN $3 ELSE 0 END),
    unread_social = GREATEST(0, unread_social - CASE WHEN $2 = 'SOCIAL' THEN $3 ELSE 0 END),
    unread_system = GREATEST(0, unread_system - CASE WHEN $2 = 'SYSTEM' THEN $3 ELSE 0 END),
    unread_security = GREATEST(0, unread_security - CASE WHEN $2 = 'SECURITY' THEN $3 ELSE 0 END),
    updated_at = $4
WHERE recipient_id = $1
