package postgres

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func (s *Store) ListDeliveries(ctx context.Context, query ports.ListDeliveriesQuery) (ports.DeliveryPage, error) {
	limit := query.Size
	if limit <= 0 {
		limit = 20
	}
	var cursorCreatedAt any
	var cursorPublicID string
	if query.Cursor != "" {
		createdAt, publicID, ok := decodeDeliveryCursor(query.Cursor)
		if !ok {
			return ports.DeliveryPage{}, ports.ErrNotificationNotFound
		}
		cursorCreatedAt = createdAt
		cursorPublicID = publicID
	}
	rows, err := s.db.QueryContext(ctx, listDeliveriesSQL, query.RecipientID, query.Channel, query.Status, cursorCreatedAt, cursorPublicID, limit+1)
	if err != nil {
		return ports.DeliveryPage{}, fmt.Errorf("list notification deliveries: %w", err)
	}
	defer rows.Close()

	items := make([]ports.Delivery, 0, limit)
	for rows.Next() {
		item, err := scanDelivery(rows)
		if err != nil {
			return ports.DeliveryPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ports.DeliveryPage{}, fmt.Errorf("iterate notification deliveries: %w", err)
	}
	page := ports.DeliveryPage{Items: items}
	if len(items) > limit {
		page.HasMore = true
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeDeliveryCursor(last.CreatedAt, last.DeliveryID)
	}
	return page, nil
}

func (s *Store) RetryDelivery(ctx context.Context, input ports.RetryDeliveryInput) (ports.DeliveryRetryResult, error) {
	var result ports.DeliveryRetryResult
	err := s.db.QueryRowContext(ctx, retryDeliverySQL, input.DeliveryID, input.RequesterID, input.IsAdmin, input.RetriedAt).Scan(
		&result.PublicID,
		&result.RecipientID,
		&result.Status,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.DeliveryRetryResult{}, ports.ErrNotificationNotFound
	}
	if err != nil {
		return ports.DeliveryRetryResult{}, fmt.Errorf("retry notification delivery: %w", err)
	}
	result.Retried = true
	return result, nil
}

type deliveryScanner interface {
	Scan(dest ...any) error
}

func scanDelivery(scanner deliveryScanner) (ports.Delivery, error) {
	var item ports.Delivery
	var notificationID sql.NullString
	var nextRetryAt sql.NullTime
	if err := scanner.Scan(
		&item.DeliveryID,
		&item.RecipientID,
		&notificationID,
		&item.Channel,
		&item.NotificationType,
		&item.Status,
		&item.Provider,
		&item.AttemptCount,
		&item.LastErrorCode,
		&nextRetryAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return ports.Delivery{}, fmt.Errorf("scan notification delivery: %w", err)
	}
	if notificationID.Valid {
		item.NotificationID = &notificationID.String
	}
	if nextRetryAt.Valid {
		item.NextRetryAt = &nextRetryAt.Time
	}
	return item, nil
}

func encodeDeliveryCursor(createdAt time.Time, publicID string) string {
	payload := createdAt.UTC().Format(time.RFC3339Nano) + "|" + publicID
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeDeliveryCursor(cursor string) (time.Time, string, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return time.Time{}, "", false
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return time.Time{}, "", false
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false
	}
	return createdAt.UTC(), parts[1], true
}
