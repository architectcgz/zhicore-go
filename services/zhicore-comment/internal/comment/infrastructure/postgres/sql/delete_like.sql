
DELETE FROM comment_likes
WHERE comment_id = $1
  AND user_id = $2