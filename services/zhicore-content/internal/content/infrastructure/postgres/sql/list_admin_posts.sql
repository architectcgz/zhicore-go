SELECT
    p.public_id AS post_id,
    p.owner_id AS author_id,
    p.owner_display_name AS author_name,
    p.owner_avatar_file_id AS author_avatar_file_id,
    CASE WHEN p.status = 'PUBLISHED' THEN p.published_title ELSE p.draft_title END AS title,
    CASE WHEN p.status = 'PUBLISHED' THEN p.published_summary ELSE p.draft_summary END AS summary,
    CASE WHEN p.status = 'PUBLISHED' THEN p.published_cover_file_id ELSE p.draft_cover_file_id END AS cover_file_id,
    p.status,
    p.post_version,
    p.published_at,
    p.created_at,
    p.updated_at,
    COALESCE(s.view_count, 0) AS view_count,
    COALESCE(s.like_count, 0) AS like_count,
    COALESCE(s.favorite_count, 0) AS favorite_count,
    COALESCE(s.comment_count, 0) AS comment_count,
    COUNT(*) OVER() AS total_count
FROM posts AS p
LEFT JOIN post_stats AS s ON s.post_id = p.id
WHERE ($1 = '' OR p.status = $1)
  AND ($2 = 0 OR p.owner_id = $2)
ORDER BY p.updated_at DESC, p.public_id DESC
LIMIT $3 OFFSET $4
