
UPDATE comment_hot_rank
SET visible = FALSE,
    updated_at = $2
WHERE comment_id = $1;
UPDATE comment_recommended_rank
SET visible = FALSE,
    updated_at = $2
WHERE comment_id = $1