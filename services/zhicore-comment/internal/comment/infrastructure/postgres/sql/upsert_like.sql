
INSERT INTO comment_likes (comment_id, user_id, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (comment_id, user_id) DO NOTHING