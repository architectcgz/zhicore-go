package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func (s *Store) CreateInteractionNotification(ctx context.Context, input ports.CreateInteractionNotificationInput) (ports.CreateInteractionNotificationResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ports.CreateInteractionNotificationResult{}, fmt.Errorf("begin notification interaction transaction: %w", err)
	}
	defer tx.Rollback()

	if err := insertConsumedEvent(ctx, tx, input.Event); errors.Is(err, sql.ErrNoRows) {
		if commitErr := tx.Commit(); commitErr != nil {
			return ports.CreateInteractionNotificationResult{}, fmt.Errorf("commit duplicate notification event: %w", commitErr)
		}
		return ports.CreateInteractionNotificationResult{}, ports.ErrDuplicateConsumedEvent
	} else if err != nil {
		return ports.CreateInteractionNotificationResult{}, err
	}

	notificationID, err := s.nextNotificationID(ctx, tx)
	if err != nil {
		return ports.CreateInteractionNotificationResult{}, err
	}
	publicID, err := s.encodePublicID(notificationID)
	if err != nil {
		return ports.CreateInteractionNotificationResult{}, err
	}

	insertedID, created, err := insertNotification(ctx, tx, notificationID, publicID, input)
	if err != nil {
		return ports.CreateInteractionNotificationResult{}, err
	}
	if created {
		if err := upsertGroupState(ctx, tx, insertedID, input); err != nil {
			return ports.CreateInteractionNotificationResult{}, err
		}
	}
	if err := markConsumedEvent(ctx, tx, input.Event.EventID, input.CreatedAt); err != nil {
		return ports.CreateInteractionNotificationResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ports.CreateInteractionNotificationResult{}, fmt.Errorf("commit notification interaction transaction: %w", err)
	}
	return ports.CreateInteractionNotificationResult{Created: created, NotificationID: insertedID, PublicID: publicID}, nil
}

func insertConsumedEvent(ctx context.Context, tx *sql.Tx, event ports.ConsumedEventMetadata) error {
	return tx.QueryRowContext(ctx, `
INSERT INTO consumed_events (event_id, event_type, routing_key, consumer_name, payload_hash, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (event_id) DO NOTHING
RETURNING event_id`, event.EventID, event.EventType, event.RoutingKey, event.ConsumerName, event.PayloadHash, event.ExpiresAt).Scan(new(string))
}

func (s *Store) nextNotificationID(ctx context.Context, tx *sql.Tx) (int64, error) {
	var id int64
	if err := tx.QueryRowContext(ctx, `SELECT nextval(pg_get_serial_sequence('notifications', 'id'))`).Scan(&id); err != nil {
		return 0, fmt.Errorf("allocate notification id: %w", err)
	}
	return id, nil
}

func (s *Store) encodePublicID(id int64) (string, error) {
	if s.codec == nil {
		return "", fmt.Errorf("notification public id codec is required")
	}
	publicID, err := s.codec.Encode(uint64(id))
	if err != nil {
		return "", fmt.Errorf("encode notification public id: %w", err)
	}
	return publicID, nil
}

func insertNotification(ctx context.Context, tx *sql.Tx, id int64, publicID string, input ports.CreateInteractionNotificationInput) (int64, bool, error) {
	var insertedID int64
	err := tx.QueryRowContext(ctx, `
INSERT INTO notifications (
    id, public_id, recipient_id, actor_id, category, notification_type, event_code, importance,
    target_type, target_id, source_event_id, dedupe_key, group_key, title, content, payload,
    occurred_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13, $14, $15, $16,
    $17, $18, $18
)
ON CONFLICT DO NOTHING
RETURNING id`,
		id,
		publicID,
		input.RecipientID,
		nullableInt64(input.ActorID),
		input.Category,
		input.NotificationType,
		input.EventCode,
		input.Importance,
		input.TargetType,
		input.TargetID,
		input.SourceEventID,
		input.DedupeKey,
		input.GroupKey,
		input.Title,
		input.Content,
		input.Payload,
		input.OccurredAt,
		input.CreatedAt,
	).Scan(&insertedID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("insert notification interaction: %w", err)
	}
	return insertedID, true, nil
}

func upsertGroupState(ctx context.Context, tx *sql.Tx, notificationID int64, input ports.CreateInteractionNotificationInput) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO notification_group_state (
    recipient_id, group_key, notification_type, category, target_type, target_id,
    latest_notification_id, total_count, unread_count, latest_time, latest_content,
    latest_actor_ids, aggregated_content, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, 1, 1, $8, $9,
    CASE WHEN $10::BIGINT IS NULL THEN '{}'::BIGINT[] ELSE ARRAY[$10::BIGINT] END,
    $11, $12, $12
)
ON CONFLICT (recipient_id, group_key) DO UPDATE SET
    latest_notification_id = EXCLUDED.latest_notification_id,
    total_count = notification_group_state.total_count + 1,
    unread_count = notification_group_state.unread_count + 1,
    latest_time = EXCLUDED.latest_time,
    latest_content = EXCLUDED.latest_content,
    latest_actor_ids = CASE
        WHEN $10::BIGINT IS NULL THEN notification_group_state.latest_actor_ids
        ELSE (ARRAY_PREPEND($10::BIGINT, ARRAY_REMOVE(notification_group_state.latest_actor_ids, $10::BIGINT)))[1:5]
    END,
    aggregated_content = EXCLUDED.aggregated_content,
    updated_at = EXCLUDED.updated_at`,
		input.RecipientID,
		input.GroupKey,
		input.NotificationType,
		input.Category,
		input.TargetType,
		input.TargetID,
		notificationID,
		input.OccurredAt,
		input.Content,
		nullableInt64(input.ActorID),
		input.Payload,
		input.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert notification group state: %w", err)
	}
	return nil
}

func markConsumedEvent(ctx context.Context, tx *sql.Tx, eventID string, consumedAt time.Time) error {
	if _, err := tx.ExecContext(ctx, `
UPDATE consumed_events
SET status = 'CONSUMED', consumed_at = $2, updated_at = $2
WHERE event_id = $1`, eventID, consumedAt); err != nil {
		return fmt.Errorf("mark consumed notification event: %w", err)
	}
	return nil
}

func nullableInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}
