
UPDATE comment_stats
SET reply_count = reply_count + 1,
    updated_at = $2
WHERE comment_id = $1