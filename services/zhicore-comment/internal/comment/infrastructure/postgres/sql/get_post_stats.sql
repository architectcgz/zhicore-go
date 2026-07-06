
SELECT post_id, total_comments, total_top_level_comments
FROM comment_post_stats
WHERE post_id = $1