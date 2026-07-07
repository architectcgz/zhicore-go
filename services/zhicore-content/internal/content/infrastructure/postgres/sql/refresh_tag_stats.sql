INSERT INTO tag_stats (tag_id, post_count, updated_at)
SELECT
    t.id,
    COUNT(p.id) AS post_count,
    $2
FROM tags AS t
LEFT JOIN post_tags AS pt ON pt.tag_id = t.id
LEFT JOIN posts AS p ON p.id = pt.post_id
    AND p.status = 'PUBLISHED'
    AND p.deleted_at IS NULL
WHERE t.id = ANY($1::BIGINT[])
GROUP BY t.id
ON CONFLICT (tag_id) DO UPDATE
SET post_count = EXCLUDED.post_count,
    updated_at = EXCLUDED.updated_at
