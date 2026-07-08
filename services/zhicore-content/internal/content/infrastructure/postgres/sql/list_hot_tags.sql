SELECT
    t.id,
    t.public_id,
    t.name,
    t.slug,
    COALESCE(ts.post_count, 0) AS post_count
FROM tags AS t
JOIN tag_stats AS ts ON ts.tag_id = t.id
WHERE t.status = 'ACTIVE'
  AND ts.post_count > 0
ORDER BY ts.post_count DESC, t.slug ASC
LIMIT $1
