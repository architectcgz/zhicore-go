
SELECT author_id
FROM comments
WHERE post_id = $1
  AND id = $2
  AND status = 'NORMAL'