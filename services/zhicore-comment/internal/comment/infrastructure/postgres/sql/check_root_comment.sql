
SELECT TRUE
FROM comments
WHERE post_id = $1
  AND id = $2
  AND root_id IS NULL
  AND parent_id IS NULL
  AND status = 'NORMAL'