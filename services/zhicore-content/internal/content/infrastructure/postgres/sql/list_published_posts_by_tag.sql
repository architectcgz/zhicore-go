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
FROM tags AS t
JOIN post_tags AS pt ON pt.tag_id = t.id
JOIN posts AS p ON p.id = pt.post_id
LEFT JOIN post_stats AS s ON s.post_id = p.id
WHERE t.slug = $1
  AND t.status = 'ACTIVE'
  AND p.status = 'PUBLISHED'
  AND p.deleted_at IS NULL
  AND ($2::TIMESTAMPTZ IS NULL OR (p.published_at, p.public_id) < ($2, $3))
ORDER BY p.published_at DESC, p.public_id DESC
LIMIT $4
