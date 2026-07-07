SELECT
    t.id,
    t.public_id,
    t.name,
    t.slug,
    COALESCE(ts.post_count, 0) AS post_count
FROM posts AS p
JOIN post_tags AS pt ON pt.post_id = p.id
JOIN tags AS t ON t.id = pt.tag_id
LEFT JOIN tag_stats AS ts ON ts.tag_id = t.id
WHERE p.public_id = $1
  AND p.status = 'PUBLISHED'
  AND p.deleted_at IS NULL
  AND t.status = 'ACTIVE'
ORDER BY pt.position ASC, t.slug ASC
