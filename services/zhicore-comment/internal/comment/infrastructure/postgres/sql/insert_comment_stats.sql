
INSERT INTO comment_stats (comment_id, like_count, reply_count, updated_at)
VALUES ($1, 0, 0, $2)
ON CONFLICT (comment_id) DO NOTHING