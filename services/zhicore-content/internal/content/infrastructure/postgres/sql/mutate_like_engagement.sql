WITH target_post AS (
    SELECT id, public_id, owner_id, post_version
    FROM posts
    WHERE public_id = $1
      AND status = 'PUBLISHED'
    FOR UPDATE
),
inserted AS (
    INSERT INTO post_likes (post_id, user_id, created_at)
    SELECT id, $2, $3
    FROM target_post
    ON CONFLICT (post_id, user_id) DO NOTHING
    RETURNING post_id
),
delta AS (
    SELECT EXISTS (SELECT 1 FROM inserted) AS changed
),
updated_stats AS (
    UPDATE post_stats AS ps
    SET like_count = ps.like_count + CASE WHEN delta.changed THEN 1 ELSE 0 END,
        updated_at = CASE WHEN delta.changed THEN $3 ELSE ps.updated_at END
    FROM target_post, delta
    WHERE ps.post_id = target_post.id
    RETURNING ps.view_count, ps.like_count, ps.favorite_count, ps.comment_count
),
updated_post AS (
    UPDATE posts AS p
    SET post_version = p.post_version + CASE WHEN delta.changed THEN 1 ELSE 0 END,
        updated_at = CASE WHEN delta.changed THEN $3 ELSE p.updated_at END
    FROM target_post, delta
    WHERE p.id = target_post.id
    RETURNING p.post_version
)
SELECT
    target_post.id AS post_internal_id,
    target_post.public_id AS post_id,
    target_post.owner_id AS author_id,
    $2::BIGINT AS actor_id,
    delta.changed AS changed,
    TRUE AS liked,
    EXISTS (
        SELECT 1
        FROM post_favorites
        WHERE post_id = target_post.id
          AND user_id = $2
    ) AS favorited,
    updated_post.post_version AS aggregate_version,
    updated_stats.view_count,
    updated_stats.like_count,
    updated_stats.favorite_count,
    updated_stats.comment_count
FROM target_post, delta, updated_stats, updated_post;
