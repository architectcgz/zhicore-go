
UPDATE posts
SET draft_body_id = $1,
    draft_body_hash = $2,
    draft_size_bytes = $3,
    draft_plain_text_length = $4,
    post_version = post_version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE public_id = $5
  AND owner_id = $6
  AND post_version = $7
  AND COALESCE(draft_body_id, '') = $8
  AND COALESCE(draft_body_hash, '') = $9
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