-- Destructive rollback: this drops User profile and outbox data.
-- Run only against disposable data or after an explicit backup/restore decision.
BEGIN;

DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS user_blocks;
DROP TABLE IF EXISTS user_follow_stats;
DROP TABLE IF EXISTS user_follows;
DROP TABLE IF EXISTS users;

COMMIT;
