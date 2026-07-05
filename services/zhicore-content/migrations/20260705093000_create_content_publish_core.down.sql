-- Destructive rollback: this drops Content post metadata, stats,
-- outbox, internal event tasks, and body cleanup / repair task records.
-- Run only against disposable data or after an explicit backup/restore decision.
BEGIN;

DROP TABLE IF EXISTS content_body_repair_tasks;
DROP TABLE IF EXISTS content_body_cleanup_tasks;
DROP TABLE IF EXISTS domain_event_tasks;
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS post_stats;
DROP TABLE IF EXISTS posts;

COMMIT;
