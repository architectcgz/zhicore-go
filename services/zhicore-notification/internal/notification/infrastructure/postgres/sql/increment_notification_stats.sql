INSERT INTO notification_stats (
    recipient_id,
    unread_total,
    unread_interaction,
    unread_content,
    unread_social,
    unread_system,
    unread_security,
    created_at,
    updated_at
)
VALUES (
    $1,
    1,
    CASE WHEN $2 = 'INTERACTION' THEN 1 ELSE 0 END,
    CASE WHEN $2 = 'CONTENT' THEN 1 ELSE 0 END,
    CASE WHEN $2 = 'SOCIAL' THEN 1 ELSE 0 END,
    CASE WHEN $2 = 'SYSTEM' THEN 1 ELSE 0 END,
    CASE WHEN $2 = 'SECURITY' THEN 1 ELSE 0 END,
    $3,
    $3
)
ON CONFLICT (recipient_id) DO UPDATE
SET unread_total = notification_stats.unread_total + 1,
    unread_interaction = notification_stats.unread_interaction + CASE WHEN $2 = 'INTERACTION' THEN 1 ELSE 0 END,
    unread_content = notification_stats.unread_content + CASE WHEN $2 = 'CONTENT' THEN 1 ELSE 0 END,
    unread_social = notification_stats.unread_social + CASE WHEN $2 = 'SOCIAL' THEN 1 ELSE 0 END,
    unread_system = notification_stats.unread_system + CASE WHEN $2 = 'SYSTEM' THEN 1 ELSE 0 END,
    unread_security = notification_stats.unread_security + CASE WHEN $2 = 'SECURITY' THEN 1 ELSE 0 END,
    updated_at = EXCLUDED.updated_at
