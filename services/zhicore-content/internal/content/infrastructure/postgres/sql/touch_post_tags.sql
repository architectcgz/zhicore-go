UPDATE posts
SET post_version = post_version + 1,
    updated_at = $1
WHERE public_id = $2
  AND owner_id = $3
  AND post_version = $4
RETURNING post_version
