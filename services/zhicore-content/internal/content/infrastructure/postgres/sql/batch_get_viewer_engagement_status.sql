WITH requested AS (
    SELECT unnest($2::TEXT[]) AS public_id
),
published_posts AS (
    SELECT p.id, p.public_id
    FROM posts AS p
    JOIN requested ON requested.public_id = p.public_id
    WHERE p.status = 'PUBLISHED'
)
SELECT
    published_posts.public_id AS post_id,
    EXISTS (
        SELECT 1
        FROM post_likes
        WHERE post_id = published_posts.id
          AND user_id = $1
    ) AS liked,
    EXISTS (
        SELECT 1
        FROM post_favorites
        WHERE post_id = published_posts.id
          AND user_id = $1
    ) AS favorited
FROM published_posts;
