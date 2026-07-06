
UPDATE posts
SET status = 'PUBLISHED',
    published_title = draft_title,
    published_summary = draft_summary,
    published_cover_file_id = draft_cover_file_id,
    published_body_id = $1,
    published_body_hash = $2,
    published_plain_text_length = $3,
    published_at = $4,
    draft_title = NULL,
    draft_summary = NULL,
    draft_cover_file_id = NULL,
    draft_body_id = NULL,
    draft_body_hash = NULL,
    draft_size_bytes = NULL,
    draft_plain_text_length = NULL,
    post_version = post_version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE public_id = $5
  AND owner_id = $6
  AND post_version = $7
  AND COALESCE(draft_body_id, '') = $8
  AND COALESCE(draft_body_hash, '') = $9
  AND status = 'DRAFT'
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