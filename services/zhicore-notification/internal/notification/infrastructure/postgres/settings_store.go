package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func (s *Store) GetNotificationPreferences(ctx context.Context, userID int64) (ports.NotificationPreferences, error) {
	rows, err := s.db.QueryContext(ctx, getNotificationPreferencesSQL, userID)
	if err != nil {
		return ports.NotificationPreferences{}, fmt.Errorf("get notification preferences: %w", err)
	}
	defer rows.Close()

	result := ports.NotificationPreferences{UserID: userID}
	for rows.Next() {
		var item ports.NotificationPreference
		if err := rows.Scan(&item.NotificationType, &item.Channel, &item.Enabled); err != nil {
			return ports.NotificationPreferences{}, fmt.Errorf("scan notification preference: %w", err)
		}
		result.Preferences = append(result.Preferences, item)
	}
	if err := rows.Err(); err != nil {
		return ports.NotificationPreferences{}, fmt.Errorf("iterate notification preferences: %w", err)
	}
	return result, nil
}

func (s *Store) SaveNotificationPreferences(ctx context.Context, input ports.SaveNotificationPreferencesInput) (ports.NotificationPreferences, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ports.NotificationPreferences{}, fmt.Errorf("begin notification preference transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, deleteNotificationPreferencesSQL, input.UserID); err != nil {
		return ports.NotificationPreferences{}, fmt.Errorf("replace notification preferences: %w", err)
	}
	for _, item := range input.Preferences {
		if _, err := tx.ExecContext(ctx, insertNotificationPreferenceSQL, input.UserID, item.NotificationType, item.Channel, item.Enabled, input.UpdatedAt); err != nil {
			return ports.NotificationPreferences{}, fmt.Errorf("insert notification preference: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return ports.NotificationPreferences{}, fmt.Errorf("commit notification preference transaction: %w", err)
	}
	return ports.NotificationPreferences{UserID: input.UserID, Preferences: input.Preferences}, nil
}

func (s *Store) GetNotificationDND(ctx context.Context, userID int64) (ports.NotificationDND, error) {
	result := ports.NotificationDND{UserID: userID}
	var categories pq.StringArray
	var channels pq.StringArray
	err := s.db.QueryRowContext(ctx, getNotificationDNDSQL, userID).Scan(
		&result.Enabled,
		&result.StartTime,
		&result.EndTime,
		&result.Timezone,
		&categories,
		&channels,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.NotificationDND{UserID: userID, Enabled: false, StartTime: "22:00", EndTime: "07:00", Timezone: "UTC"}, nil
	}
	if err != nil {
		return ports.NotificationDND{}, fmt.Errorf("get notification dnd: %w", err)
	}
	result.Categories = []string(categories)
	result.Channels = []string(channels)
	return result, nil
}

func (s *Store) SaveNotificationDND(ctx context.Context, input ports.SaveNotificationDNDInput) (ports.NotificationDND, error) {
	result := ports.NotificationDND{UserID: input.UserID}
	var categories pq.StringArray
	var channels pq.StringArray
	err := s.db.QueryRowContext(ctx, upsertNotificationDNDSQL,
		input.UserID,
		input.Enabled,
		input.StartTime,
		input.EndTime,
		input.Timezone,
		pq.Array(input.Categories),
		pq.Array(input.Channels),
		input.UpdatedAt,
	).Scan(
		&result.Enabled,
		&result.StartTime,
		&result.EndTime,
		&result.Timezone,
		&categories,
		&channels,
	)
	if err != nil {
		return ports.NotificationDND{}, fmt.Errorf("upsert notification dnd: %w", err)
	}
	result.Categories = []string(categories)
	result.Channels = []string(channels)
	return result, nil
}

func (s *Store) GetAuthorSubscription(ctx context.Context, input ports.GetAuthorSubscriptionInput) (ports.AuthorSubscription, error) {
	result := ports.AuthorSubscription{UserID: input.UserID, AuthorID: input.AuthorID}
	err := s.db.QueryRowContext(ctx, getAuthorSubscriptionSQL, input.UserID, input.AuthorID).Scan(
		&result.Level,
		&result.InAppEnabled,
		&result.WebsocketEnabled,
		&result.EmailEnabled,
		&result.DigestEnabled,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.AuthorSubscription{UserID: input.UserID, AuthorID: input.AuthorID, Level: "ALL", InAppEnabled: true, WebsocketEnabled: true, DigestEnabled: true}, nil
	}
	if err != nil {
		return ports.AuthorSubscription{}, fmt.Errorf("get author subscription: %w", err)
	}
	return result, nil
}

func (s *Store) SaveAuthorSubscription(ctx context.Context, input ports.SaveAuthorSubscriptionInput) (ports.AuthorSubscription, error) {
	result := ports.AuthorSubscription{UserID: input.UserID, AuthorID: input.AuthorID}
	err := s.db.QueryRowContext(ctx, upsertAuthorSubscriptionSQL,
		input.UserID,
		input.AuthorID,
		input.Level,
		input.InAppEnabled,
		input.WebsocketEnabled,
		input.EmailEnabled,
		input.DigestEnabled,
		input.UpdatedAt,
	).Scan(
		&result.Level,
		&result.InAppEnabled,
		&result.WebsocketEnabled,
		&result.EmailEnabled,
		&result.DigestEnabled,
	)
	if err != nil {
		return ports.AuthorSubscription{}, fmt.Errorf("upsert author subscription: %w", err)
	}
	return result, nil
}
