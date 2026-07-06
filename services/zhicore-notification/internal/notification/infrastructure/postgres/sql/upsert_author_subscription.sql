INSERT INTO notification_author_subscription (
    user_id,
    author_id,
    level,
    in_app_enabled,
    websocket_enabled,
    email_enabled,
    digest_enabled,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
ON CONFLICT (user_id, author_id) DO UPDATE
SET level = EXCLUDED.level,
    in_app_enabled = EXCLUDED.in_app_enabled,
    websocket_enabled = EXCLUDED.websocket_enabled,
    email_enabled = EXCLUDED.email_enabled,
    digest_enabled = EXCLUDED.digest_enabled,
    updated_at = EXCLUDED.updated_at
RETURNING level, in_app_enabled, websocket_enabled, email_enabled, digest_enabled
