INSERT INTO notification_campaign (
    source_event_id,
    campaign_type,
    author_id,
    post_id,
    object_type,
    object_id,
    title,
    excerpt,
    payload,
    published_at,
    status,
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
    'PLANNED',
    $11,
    $11
)
ON CONFLICT (source_event_id) DO NOTHING
RETURNING id;
