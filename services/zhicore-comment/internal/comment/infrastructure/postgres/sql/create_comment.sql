
INSERT INTO comments (
    post_id,
    content_internal_id,
    author_id,
    root_id,
    parent_id,
    content,
    image_file_ids,
    voice_file_id,
    voice_duration,
    status,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING
    id,
    post_id,
    content_internal_id,
    author_id,
    root_id,
    parent_id,
    content,
    image_file_ids,
    voice_file_id,
    voice_duration,
    status,
    created_at,
    updated_at