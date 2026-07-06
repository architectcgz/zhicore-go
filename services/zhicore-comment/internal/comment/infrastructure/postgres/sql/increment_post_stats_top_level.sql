
INSERT INTO comment_post_stats (post_id, total_comments, total_top_level_comments, updated_at)
VALUES ($1, 1, 1, $2)
ON CONFLICT (post_id) DO UPDATE
SET total_comments = comment_post_stats.total_comments + 1,
    total_top_level_comments = comment_post_stats.total_top_level_comments + 1,
    updated_at = EXCLUDED.updated_at