BEGIN;
DROP INDEX IF EXISTS ix_notifications_recipient_group_id;
DROP INDEX IF EXISTS ux_notification_group_state_recipient_group_id;
ALTER TABLE notification_group_state DROP COLUMN group_id;
ALTER TABLE notifications DROP COLUMN actor_avatar_url;
ALTER TABLE notifications DROP COLUMN actor_display_name;
ALTER TABLE notifications DROP COLUMN actor_public_id;
ALTER TABLE notifications DROP COLUMN group_id;
COMMIT;
