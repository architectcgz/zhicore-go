SELECT EXISTS (
    SELECT 1
    FROM posts
    WHERE public_id = $1
      AND status = 'PUBLISHED'
      AND deleted_at IS NULL
)
