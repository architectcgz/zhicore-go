package postgres

import (
	"context"
	"crypto/md5"
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
		if err := incrementNotificationStats(ctx, tx, input); err != nil {
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
	return tx.QueryRowContext(ctx, insertConsumedEventSQL, event.EventID, event.EventType, event.RoutingKey, event.ConsumerName, event.PayloadHash, event.ExpiresAt).Scan(new(string))
}

func (s *Store) nextNotificationID(ctx context.Context, tx *sql.Tx) (int64, error) {
	var id int64
	if err := tx.QueryRowContext(ctx, nextNotificationIDSQL).Scan(&id); err != nil {
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
	err := tx.QueryRowContext(ctx, insertInteractionNotificationSQL,
		id,
		publicID,
		input.RecipientID,
		nullableInt64(input.ActorID),
		input.ActorPublicID,
		input.ActorDisplayName,
		input.ActorAvatarURL,
		input.Category,
		input.NotificationType,
		input.EventCode,
		input.Importance,
		input.TargetType,
		input.TargetID,
		input.SourceEventID,
		input.DedupeKey,
		input.GroupKey,
		groupPublicID(input.RecipientID, input.GroupKey),
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
	_, err := tx.ExecContext(ctx, upsertInteractionNotificationGroupSQL,
		input.RecipientID,
		input.GroupKey,
		groupPublicID(input.RecipientID, input.GroupKey),
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

func groupPublicID(recipientID int64, groupKey string) string {
	sum := md5.Sum([]byte(fmt.Sprintf("%d:%s", recipientID, groupKey)))
	return fmt.Sprintf("ng%x", sum[:15])
}

func incrementNotificationStats(ctx context.Context, tx *sql.Tx, input ports.CreateInteractionNotificationInput) error {
	if _, err := tx.ExecContext(ctx, incrementNotificationStatsSQL, input.RecipientID, input.Category, input.CreatedAt); err != nil {
		return fmt.Errorf("increment notification stats unread count: %w", err)
	}
	return nil
}

func markConsumedEvent(ctx context.Context, tx *sql.Tx, eventID string, consumedAt time.Time) error {
	if _, err := tx.ExecContext(ctx, markConsumedEventSQL, eventID, consumedAt); err != nil {
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
