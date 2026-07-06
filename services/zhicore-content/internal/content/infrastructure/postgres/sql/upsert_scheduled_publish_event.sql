INSERT INTO scheduled_publish_events (
    post_id,
    public_id,
    owner_id,
    draft_body_id,
    draft_body_hash,
    scheduled_at,
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
    'PENDING',
    $7,
    $7
)
ON CONFLICT (post_id)
WHERE status = 'PENDING'
DO UPDATE SET
    draft_body_id = EXCLUDED.draft_body_id,
    draft_body_hash = EXCLUDED.draft_body_hash,
    scheduled_at = EXCLUDED.scheduled_at,
    updated_at = EXCLUDED.updated_at
