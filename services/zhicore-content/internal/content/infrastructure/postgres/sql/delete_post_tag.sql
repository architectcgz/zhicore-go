DELETE FROM post_tags
WHERE post_id = $1
  AND tag_id = $2
