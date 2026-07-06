
SELECT
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
FROM comments
WHERE post_id = $1
  AND id = $2