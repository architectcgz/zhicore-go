SELECT
    t.id,
    t.public_id,
    t.name,
    t.slug,
    COALESCE(ts.post_count, 0) AS post_count
FROM tags AS t
LEFT JOIN tag_stats AS ts ON ts.tag_id = t.id
WHERE t.slug = ANY($1::TEXT[])
  AND t.status = 'ACTIVE'
