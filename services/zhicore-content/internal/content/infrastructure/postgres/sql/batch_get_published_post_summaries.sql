
SELECT
    p.public_id,
    p.owner_id,
    p.owner_display_name,
    p.owner_avatar_file_id,
    p.published_title,
    p.published_summary,
    p.published_cover_file_id,
    p.status,
    p.post_version,
    p.published_at,
    p.created_at,
    p.updated_at,
    COALESCE(s.view_count, 0),
    COALESCE(s.like_count, 0),
    COALESCE(s.favorite_count, 0),
    COALESCE(s.comment_count, 0)
FROM posts AS p
LEFT JOIN post_stats AS s ON s.post_id = p.id
WHERE p.public_id = ANY($1)
  AND p.status = 'PUBLISHED'
  AND p.deleted_at IS NULL