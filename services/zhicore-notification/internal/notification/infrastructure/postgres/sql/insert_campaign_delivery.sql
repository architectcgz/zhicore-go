INSERT INTO notification_delivery (
    id,
    public_id,
    recipient_id,
    notification_id,
    campaign_id,
    channel,
    notification_type,
    status,
    dedupe_key,
    created_at,
    updated_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $10
)
ON CONFLICT (dedupe_key) DO NOTHING;
