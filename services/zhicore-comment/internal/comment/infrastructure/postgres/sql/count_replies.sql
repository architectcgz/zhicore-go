
SELECT COUNT(*)::BIGINT
FROM comments
WHERE post_id = $1
  AND root_id = $2
  AND status = 'NORMAL'