WITH target_post AS (
    SELECT id, public_id, owner_id, post_version
    FROM posts
    WHERE public_id = $1
      AND status = 'PUBLISHED'
),
inserted AS (
    INSERT INTO post_favorites (post_id, user_id, created_at)
    SELECT id, $2, $3
    FROM target_post
    ON CONFLICT (post_id, user_id) DO NOTHING
    RETURNING post_id
),
delta AS (
    SELECT EXISTS (SELECT 1 FROM inserted) AS changed
),
current_stats AS (
    SELECT ps.view_count, ps.like_count, ps.favorite_count, ps.comment_count
    FROM post_stats AS ps
    JOIN target_post ON target_post.id = ps.post_id
)
SELECT
    target_post.id AS post_internal_id,
    target_post.public_id AS post_id,
    target_post.owner_id AS author_id,
    $2::BIGINT AS actor_id,
    delta.changed AS changed,
    EXISTS (
        SELECT 1
        FROM post_likes
        WHERE post_id = target_post.id
          AND user_id = $2
    ) AS liked,
    TRUE AS favorited,
    target_post.post_version AS aggregate_version,
    current_stats.view_count,
    current_stats.like_count,
    current_stats.favorite_count,
    current_stats.comment_count
FROM target_post, delta, current_stats;
