
WITH RECURSIVE subtree AS (
    SELECT id
    FROM comments
    WHERE post_id = $1
      AND id = $2
    UNION ALL
    SELECT c.id
    FROM comments c
    JOIN subtree s ON c.parent_id = s.id
    WHERE c.post_id = $1
),
updated AS (
    UPDATE comments
    SET status = 'DELETED',
        deleted_by = $3,
        deleted_by_role = $4,
        delete_reason = $5,
        deleted_at = $6,
        updated_at = $6
    WHERE id IN (SELECT id FROM subtree)
      AND status = 'NORMAL'
    RETURNING id
)
SELECT COUNT(*)::BIGINT AS affected_count
FROM updated