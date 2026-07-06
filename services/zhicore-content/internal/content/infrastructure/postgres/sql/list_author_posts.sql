SELECT
    p.public_id,
    p.owner_id,
    p.owner_display_name,
    p.owner_avatar_file_id,
    CASE WHEN p.status = 'PUBLISHED' THEN p.published_title ELSE p.draft_title END AS title,
    CASE WHEN p.status = 'PUBLISHED' THEN p.published_summary ELSE p.draft_summary END AS summary,
    CASE WHEN p.status = 'PUBLISHED' THEN p.published_cover_file_id ELSE p.draft_cover_file_id END AS cover_file_id,
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
WHERE p.owner_id = $1
  AND ($2 = '' OR p.status = $2)
  AND ($3::TIMESTAMPTZ IS NULL OR (p.updated_at, p.public_id) < ($3, $4))
ORDER BY p.updated_at DESC, p.public_id DESC
LIMIT $5
