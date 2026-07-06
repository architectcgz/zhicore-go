UPDATE posts
SET draft_title = COALESCE($1, draft_title),
    draft_summary = CASE WHEN $2::TEXT IS NULL THEN draft_summary ELSE NULLIF($2, '') END,
    draft_cover_file_id = CASE WHEN $3::TEXT IS NULL THEN draft_cover_file_id ELSE NULLIF($3, '') END,
    post_version = post_version + 1,
    updated_at = $4
WHERE public_id = $5
  AND owner_id = $6
  AND post_version = $7
  AND status <> 'DELETED'
RETURNING
    id,
    public_id,
    owner_id,
    status,
    post_version,
    draft_title,
    draft_summary,
    draft_cover_file_id,
    draft_body_id,
    draft_body_hash,
    draft_size_bytes,
    draft_plain_text_length,
    published_title,
    published_summary,
    published_cover_file_id,
    published_body_id,
    published_body_hash,
    published_plain_text_length,
    published_at
