
INSERT INTO comment_hot_rank (comment_id, post_id, like_count, visible, updated_at)
VALUES ($1, $2, 0, TRUE, $3)
ON CONFLICT (comment_id) DO UPDATE
SET post_id = EXCLUDED.post_id,
    visible = TRUE,
    updated_at = EXCLUDED.updated_at