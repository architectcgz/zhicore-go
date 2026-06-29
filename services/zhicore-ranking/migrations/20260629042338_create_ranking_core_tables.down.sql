-- Data loss warning:
-- This rollback drops Ranking ledger, bucket, projection, state, and period score tables.
-- Re-applying the migration cannot restore dropped ranking history or materialized state.
BEGIN;

DROP TABLE IF EXISTS ranking_period_score;
DROP TABLE IF EXISTS ranking_projection_event_inbox;
DROP TABLE IF EXISTS ranking_post_state;
DROP TABLE IF EXISTS ranking_delta_bucket;
DROP TABLE IF EXISTS ranking_event_ledger;

COMMIT;
