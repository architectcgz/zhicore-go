
SELECT EXISTS (
    SELECT 1
    FROM posts
    WHERE published_body_id = $1
       OR draft_body_id = $1
)