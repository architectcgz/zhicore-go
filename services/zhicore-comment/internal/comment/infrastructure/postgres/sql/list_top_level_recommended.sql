
SELECT
    c.id,
    c.post_id,
    c.content_internal_id,
    c.author_id,
    c.root_id,
    c.parent_id,
    c.content,
    c.image_file_ids,
    c.voice_file_id,
    c.voice_duration,
    c.status,
    c.created_at,
    c.updated_at,
    s.like_count,
    s.reply_count
FROM comment_recommended_rank r
JOIN comments c ON c.id = r.comment_id
JOIN comment_stats s ON s.comment_id = c.id
WHERE r.post_id = $1
  AND r.visible = TRUE
  AND c.status = 'NORMAL'
ORDER BY r.recommended_score DESC, c.id DESC
LIMIT $2 OFFSET $3