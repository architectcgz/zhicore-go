
SELECT
    id,
    public_id,
    status,
    published_body_id,
    published_body_hash,
    published_plain_text_length
FROM posts
WHERE public_id = $1