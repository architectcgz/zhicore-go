SELECT
    t.id,
    t.public_id,
    t.name,
    t.slug,
    COALESCE(ts.post_count, 0) AS post_count
FROM post_tags AS pt
JOIN tags AS t ON t.id = pt.tag_id
LEFT JOIN tag_stats AS ts ON ts.tag_id = t.id
WHERE pt.post_id = $1
  AND t.status = 'ACTIVE'
ORDER BY pt.position ASC, t.slug ASC
