
SELECT owner_id, status, post_version, draft_body_id, draft_body_hash
FROM posts
WHERE public_id = $1
FOR UPDATE