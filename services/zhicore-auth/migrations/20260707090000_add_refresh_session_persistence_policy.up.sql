BEGIN;

ALTER TABLE auth_refresh_sessions
    ADD COLUMN persistence_policy VARCHAR(32) NULL;

UPDATE auth_refresh_sessions
SET persistence_policy = 'STANDARD'
WHERE persistence_policy IS NULL;

ALTER TABLE auth_refresh_sessions
    ALTER COLUMN persistence_policy SET NOT NULL,
    ADD CONSTRAINT auth_refresh_sessions_persistence_policy_check
        CHECK (persistence_policy IN ('STANDARD', 'REMEMBERED'));

COMMIT;
