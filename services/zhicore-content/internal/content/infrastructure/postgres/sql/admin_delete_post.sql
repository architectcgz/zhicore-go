UPDATE posts
SET status = 'DELETED',
    post_version = post_version + 1,
    updated_at = $2,
    deleted_at = $2
WHERE public_id = $1
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
