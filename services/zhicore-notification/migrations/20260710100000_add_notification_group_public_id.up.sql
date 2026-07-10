BEGIN;

ALTER TABLE notifications ADD COLUMN group_id VARCHAR(64);
ALTER TABLE notifications ADD COLUMN actor_public_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN actor_display_name VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN actor_avatar_url TEXT NULL;
ALTER TABLE notification_group_state ADD COLUMN group_id VARCHAR(64);

-- The deterministic value keeps repaired state and inbox fallback on the same public group.
UPDATE notifications SET group_id = 'ng' || substr(md5(recipient_id::text || ':' || group_key), 1, 30) WHERE group_id IS NULL;
UPDATE notification_group_state SET group_id = 'ng' || substr(md5(recipient_id::text || ':' || group_key), 1, 30) WHERE group_id IS NULL;

ALTER TABLE notifications ALTER COLUMN group_id SET NOT NULL;
ALTER TABLE notification_group_state ALTER COLUMN group_id SET NOT NULL;
CREATE UNIQUE INDEX ux_notification_group_state_recipient_group_id ON notification_group_state (recipient_id, group_id);
CREATE INDEX ix_notifications_recipient_group_id ON notifications (recipient_id, group_id);

COMMIT;
