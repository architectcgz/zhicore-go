
INSERT INTO posts (
    public_id,
    owner_id,
    owner_display_name,
    owner_avatar_file_id,
    owner_profile_version,
    status,
    post_version,
    draft_title,
    draft_summary,
    draft_cover_file_id,
    draft_body_id,
    draft_body_hash,
    draft_size_bytes,
    draft_plain_text_length
)
VALUES ($1, $2, $3, $4, $5, 'DRAFT', 1, $6, $7, $8, $9, $10, $11, $12)
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