
UPDATE comment_stats
SET reply_count = GREATEST(0, reply_count - $2),
    updated_at = $3
WHERE comment_id = $1