
INSERT INTO post_stats (post_id, updated_at)
VALUES ($1, $2)
ON CONFLICT (post_id) DO NOTHING