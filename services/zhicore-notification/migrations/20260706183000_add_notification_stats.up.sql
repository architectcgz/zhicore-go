BEGIN;

CREATE TABLE notification_stats (
    recipient_id BIGINT PRIMARY KEY,
    unread_total BIGINT NOT NULL DEFAULT 0,
    unread_interaction BIGINT NOT NULL DEFAULT 0,
    unread_content BIGINT NOT NULL DEFAULT 0,
    unread_social BIGINT NOT NULL DEFAULT 0,
    unread_system BIGINT NOT NULL DEFAULT 0,
    unread_security BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (recipient_id > 0),
    CHECK (unread_total >= 0),
    CHECK (unread_interaction >= 0),
    CHECK (unread_content >= 0),
    CHECK (unread_social >= 0),
    CHECK (unread_system >= 0),
    CHECK (unread_security >= 0),
    CHECK (unread_total = unread_interaction + unread_content + unread_social + unread_system + unread_security)
);

COMMIT;
