INSERT INTO admin_post_audit (
    post_id,
    public_id,
    admin_user_id,
    action,
    reason,
    previous_status,
    new_status,
    occurred_at,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
