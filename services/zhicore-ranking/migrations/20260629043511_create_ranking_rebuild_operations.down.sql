-- Data loss warning:
-- This rollback drops Ranking rebuild operation audit and status records.
BEGIN;

DROP TABLE IF EXISTS ranking_rebuild_operation;

COMMIT;
