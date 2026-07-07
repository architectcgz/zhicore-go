-- Destructive rollback: this drops refresh session persistence policy metadata.
-- Run only after confirming no deployed code still depends on this column.
BEGIN;

ALTER TABLE auth_refresh_sessions
    DROP CONSTRAINT IF EXISTS auth_refresh_sessions_persistence_policy_check,
    DROP COLUMN IF EXISTS persistence_policy;

COMMIT;
