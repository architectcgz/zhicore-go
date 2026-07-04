-- Destructive rollback: this drops Auth account, password credential, role,
-- refresh session, used token, security operation, audit log, and outbox data.
-- Run only against disposable data or after an explicit backup/restore decision.
BEGIN;

DROP TABLE IF EXISTS auth_outbox_events;
DROP TABLE IF EXISTS auth_audit_logs;
DROP TABLE IF EXISTS auth_security_operations;
DROP TABLE IF EXISTS auth_used_refresh_tokens;
DROP TABLE IF EXISTS auth_refresh_sessions;
DROP TABLE IF EXISTS auth_account_roles;
DROP TABLE IF EXISTS auth_password_credentials;
DROP TABLE IF EXISTS auth_accounts;

COMMIT;
