
UPDATE comment_post_stats
SET total_comments = GREATEST(0, total_comments - $2),
    total_top_level_comments = CASE WHEN $3 THEN GREATEST(0, total_top_level_comments - 1) ELSE total_top_level_comments END,
    updated_at = $4
WHERE post_id = $1