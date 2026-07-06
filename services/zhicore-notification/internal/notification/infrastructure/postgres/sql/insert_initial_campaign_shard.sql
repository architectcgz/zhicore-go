INSERT INTO notification_campaign_shard (
    campaign_id,
    status,
    created_at,
    updated_at
) VALUES (
    $1,
    'PENDING',
    $2,
    $2
)
RETURNING id;
