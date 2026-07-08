SELECT
    t.id,
    t.public_id,
    t.name,
    t.slug,
    COALESCE(ts.post_count, 0) AS post_count
FROM tags AS t
LEFT JOIN tag_stats AS ts ON ts.tag_id = t.id
WHERE t.status = 'ACTIVE'
  AND (t.slug LIKE $1 || '%' OR lower(t.name) LIKE $1 || '%')
ORDER BY
    CASE WHEN t.slug LIKE $1 || '%' THEN 0 ELSE 1 END,
    COALESCE(ts.post_count, 0) DESC,
    t.slug ASC
LIMIT $2
