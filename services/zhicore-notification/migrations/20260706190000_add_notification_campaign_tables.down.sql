BEGIN;

ALTER TABLE notification_delivery
    DROP CONSTRAINT IF EXISTS fk_notification_delivery_campaign;

DROP TABLE IF EXISTS notification_campaign_shard;
DROP TABLE IF EXISTS notification_campaign;

COMMIT;
