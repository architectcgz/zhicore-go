SELECT tag_id
FROM post_tags
WHERE post_id = $1
ORDER BY tag_id ASC
