WITH claimed AS (
    SELECT id
    FROM domain_event_tasks
    WHERE id = $1
      AND claimed_by = $2
      AND status = 'PROCESSING'
      AND event_type = 'content.engagement.stats_delta'
    FOR UPDATE
),
updated_stats AS (
    UPDATE post_stats AS stats
    SET like_count = CASE
            WHEN $4 = 'LIKE' THEN GREATEST(0, stats.like_count + $5)
            ELSE stats.like_count
        END,
        favorite_count = CASE
            WHEN $4 = 'FAVORITE' THEN GREATEST(0, stats.favorite_count + $5)
            ELSE stats.favorite_count
        END,
        updated_at = $6
    FROM claimed
    WHERE stats.post_id = $3
      AND $4 IN ('LIKE', 'FAVORITE')
    RETURNING stats.post_id
)
UPDATE domain_event_tasks AS task
SET status = 'DONE',
    processed_at = $6,
    updated_at = $6
FROM claimed, updated_stats
WHERE task.id = claimed.id
