package postgres

const postRecordColumns = `
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
    published_at`

const insertPostSQL = `
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
RETURNING` + postRecordColumns

const insertPostStatsSQL = `
INSERT INTO post_stats (post_id, updated_at)
VALUES ($1, $2)
ON CONFLICT (post_id) DO NOTHING`

const selectPostForUpdateSQL = `
SELECT` + postRecordColumns + `
FROM posts
WHERE public_id = $1
FOR UPDATE`

const updateDraftBodySQL = `
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
RETURNING` + postRecordColumns

const publishPostSQL = `
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
RETURNING` + postRecordColumns

const classifyPostMutationMissSQL = `
SELECT owner_id, status, post_version, draft_body_id, draft_body_hash
FROM posts
WHERE public_id = $1
FOR UPDATE`

const selectPublishedBodyPointerSQL = `
SELECT
    id,
    public_id,
    status,
    published_body_id,
    published_body_hash,
    published_plain_text_length
FROM posts
WHERE public_id = $1`

const insertOutboxEventSQL = `
INSERT INTO outbox_events (
    event_id,
    event_type,
    payload_version,
    aggregate_type,
    aggregate_id,
    aggregate_version,
    payload_json,
    occurred_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

const upsertCleanupTaskSQL = `
INSERT INTO content_body_cleanup_tasks (
    post_id,
    body_id,
    task_type,
    reason,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $5)
ON CONFLICT (body_id, task_type) DO UPDATE
SET post_id = COALESCE(content_body_cleanup_tasks.post_id, EXCLUDED.post_id),
    reason = EXCLUDED.reason,
    updated_at = EXCLUDED.updated_at`

const upsertRepairTaskSQL = `
INSERT INTO content_body_repair_tasks (
    post_id,
    body_id,
    task_type,
    expected_hash,
    observed_hash,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $6)
ON CONFLICT (post_id, task_type, body_id) DO UPDATE
SET expected_hash = EXCLUDED.expected_hash,
    observed_hash = EXCLUDED.observed_hash,
    updated_at = EXCLUDED.updated_at`
