SELECT
    t.id,
    t.public_id,
    t.name,
    t.slug,
    COALESCE(ts.post_count, 0) AS post_count
FROM tags AS t
LEFT JOIN tag_stats AS ts ON ts.tag_id = t.id
WHERE t.status = 'ACTIVE'
  AND ($1::TEXT = '' OR (t.slug, t.id) > ($1, $2))
ORDER BY t.slug ASC, t.id ASC
LIMIT $3
