-- Destructive rollback: this drops Comment, stats, ranking, like, delta,
-- and outbox data. Run only against disposable data or after an explicit
-- backup/restore decision.
BEGIN;

DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS comment_recommended_rank;
DROP TABLE IF EXISTS comment_hot_rank;
DROP TABLE IF EXISTS comment_counter_deltas;
DROP TABLE IF EXISTS comment_likes;
DROP TABLE IF EXISTS comment_post_stats;
DROP TABLE IF EXISTS comment_stats;
DROP TABLE IF EXISTS comments;

COMMIT;
