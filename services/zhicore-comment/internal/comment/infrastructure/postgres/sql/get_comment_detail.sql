
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
FROM comments c
JOIN comment_stats s ON s.comment_id = c.id
WHERE c.post_id = $1
  AND c.id = $2
  AND c.status = 'NORMAL'