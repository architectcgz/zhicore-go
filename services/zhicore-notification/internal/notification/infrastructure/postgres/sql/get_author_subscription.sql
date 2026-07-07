SELECT level, in_app_enabled, websocket_enabled, email_enabled, digest_enabled
FROM notification_author_subscription
WHERE user_id = $1 AND author_id = $2
