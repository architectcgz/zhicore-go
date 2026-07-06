UPDATE posts
SET status = 'DRAFT',
    post_version = post_version + 1,
    updated_at = $4,
    deleted_at = NULL
WHERE public_id = $1
  AND owner_id = $2
  AND ($3 = 0 OR post_version = $3)
  AND status = 'DELETED'
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
