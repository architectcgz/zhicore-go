INSERT INTO notification_campaign_shard (
    campaign_id,
    audience_class,
    audience_active_since,
    status,
    created_at,
    updated_at
) VALUES (
    $1,
    $2,
    $3,
    'PENDING',
    $4,
    $4
)
RETURNING id;
