SELECT
    p.public_id,
    ps.view_count,
    ps.like_count,
    ps.favorite_count,
    ps.comment_count
FROM posts AS p
JOIN post_stats AS ps ON ps.post_id = p.id
WHERE p.public_id = $1
  AND p.status = 'PUBLISHED';
