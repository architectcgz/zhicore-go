package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/domain/preference"
	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func (s *Service) GetNotificationPreferences(ctx context.Context, query GetNotificationPreferencesQuery) (NotificationPreferencesResult, error) {
	if err := requireActor(query.Actor); err != nil {
		return NotificationPreferencesResult{}, err
	}
	if s.settings == nil {
		return NotificationPreferencesResult{}, ErrDependencyUnavailable
	}
	prefs, err := s.settings.GetNotificationPreferences(ctx, query.Actor.UserID)
	if err != nil {
		return NotificationPreferencesResult{}, mapPortsError(err)
	}
	return notificationPreferencesResult(prefs), nil
}

func (s *Service) UpdateNotificationPreferences(ctx context.Context, command UpdateNotificationPreferencesCommand) (NotificationPreferencesResult, error) {
	if err := requireActor(command.Actor); err != nil {
		return NotificationPreferencesResult{}, err
	}
	if s.settings == nil {
		return NotificationPreferencesResult{}, ErrDependencyUnavailable
	}
	now := s.clock.Now()
	input := ports.SaveNotificationPreferencesInput{UserID: command.Actor.UserID, UpdatedAt: now}
	for _, item := range command.Preferences {
		normalized, err := preference.NormalizeChannelPreference(preference.ChannelPreference(item.Channels))
		if err != nil {
			return NotificationPreferencesResult{}, ErrInvalidRequest
		}
		notificationType := strings.ToUpper(strings.TrimSpace(item.NotificationType))
		if notificationType == "" {
			return NotificationPreferencesResult{}, ErrInvalidRequest
		}
		input.Preferences = append(input.Preferences,
			ports.NotificationPreference{NotificationType: notificationType, Channel: preference.ChannelInApp, Enabled: normalized.InApp},
			ports.NotificationPreference{NotificationType: notificationType, Channel: preference.ChannelWebsocket, Enabled: normalized.Websocket},
			ports.NotificationPreference{NotificationType: notificationType, Channel: preference.ChannelEmail, Enabled: normalized.Email},
			ports.NotificationPreference{NotificationType: notificationType, Channel: preference.ChannelSMS, Enabled: normalized.SMS},
		)
	}
	result, err := s.settings.SaveNotificationPreferences(ctx, input)
	if err != nil {
		return NotificationPreferencesResult{}, mapPortsError(err)
	}
	if err := s.unread.Delete(ctx, preferenceCacheKey(command.Actor.UserID)); err != nil {
		return NotificationPreferencesResult{}, fmt.Errorf("invalidate notification preference cache: %w", err)
	}
	return notificationPreferencesResult(result), nil
}

func (s *Service) GetNotificationDND(ctx context.Context, query GetNotificationDNDQuery) (NotificationDNDResult, error) {
	if err := requireActor(query.Actor); err != nil {
		return NotificationDNDResult{}, err
	}
	if s.settings == nil {
		return NotificationDNDResult{}, ErrDependencyUnavailable
	}
	result, err := s.settings.GetNotificationDND(ctx, query.Actor.UserID)
	if err != nil {
		return NotificationDNDResult{}, mapPortsError(err)
	}
	return notificationDNDResult(result), nil
}

func (s *Service) UpdateNotificationDND(ctx context.Context, command UpdateNotificationDNDCommand) (NotificationDNDResult, error) {
	if err := requireActor(command.Actor); err != nil {
		return NotificationDNDResult{}, err
	}
	if s.settings == nil {
		return NotificationDNDResult{}, ErrDependencyUnavailable
	}
	normalized, err := preference.NormalizeDNDWindow(preference.DNDWindow{
		Enabled:    command.Enabled,
		StartTime:  command.StartTime,
		EndTime:    command.EndTime,
		Timezone:   command.Timezone,
		Categories: command.Categories,
		Channels:   command.Channels,
	})
	if err != nil {
		return NotificationDNDResult{}, ErrInvalidRequest
	}
	result, err := s.settings.SaveNotificationDND(ctx, ports.SaveNotificationDNDInput{
		UserID:     command.Actor.UserID,
		Enabled:    normalized.Enabled,
		StartTime:  normalized.StartTime,
		EndTime:    normalized.EndTime,
		Timezone:   normalized.Timezone,
		Categories: normalized.Categories,
		Channels:   normalized.Channels,
		UpdatedAt:  s.clock.Now(),
	})
	if err != nil {
		return NotificationDNDResult{}, mapPortsError(err)
	}
	if err := s.unread.Delete(ctx, dndCacheKey(command.Actor.UserID)); err != nil {
		return NotificationDNDResult{}, fmt.Errorf("invalidate notification dnd cache: %w", err)
	}
	return notificationDNDResult(result), nil
}

func (s *Service) GetAuthorSubscription(ctx context.Context, query GetAuthorSubscriptionQuery) (AuthorSubscriptionResult, error) {
	if err := requireActor(query.Actor); err != nil {
		return AuthorSubscriptionResult{}, err
	}
	if query.AuthorID <= 0 {
		return AuthorSubscriptionResult{}, ErrInvalidRequest
	}
	if s.settings == nil {
		return AuthorSubscriptionResult{}, ErrDependencyUnavailable
	}
	result, err := s.settings.GetAuthorSubscription(ctx, ports.GetAuthorSubscriptionInput{UserID: query.Actor.UserID, AuthorID: query.AuthorID})
	if err != nil {
		return AuthorSubscriptionResult{}, mapPortsError(err)
	}
	return authorSubscriptionResult(result), nil
}

func (s *Service) UpdateAuthorSubscription(ctx context.Context, command UpdateAuthorSubscriptionCommand) (AuthorSubscriptionResult, error) {
	if err := requireActor(command.Actor); err != nil {
		return AuthorSubscriptionResult{}, err
	}
	if command.AuthorID <= 0 {
		return AuthorSubscriptionResult{}, ErrInvalidRequest
	}
	if s.settings == nil {
		return AuthorSubscriptionResult{}, ErrDependencyUnavailable
	}
	normalized, err := preference.NormalizeAuthorSubscription(preference.AuthorSubscription{
		Level:            command.Level,
		InAppEnabled:     command.InAppEnabled,
		WebsocketEnabled: command.WebsocketEnabled,
		EmailEnabled:     command.EmailEnabled,
		DigestEnabled:    command.DigestEnabled,
	})
	if err != nil {
		return AuthorSubscriptionResult{}, ErrInvalidRequest
	}
	result, err := s.settings.SaveAuthorSubscription(ctx, ports.SaveAuthorSubscriptionInput{
		UserID:           command.Actor.UserID,
		AuthorID:         command.AuthorID,
		Level:            normalized.Level,
		InAppEnabled:     normalized.InAppEnabled,
		WebsocketEnabled: normalized.WebsocketEnabled,
		EmailEnabled:     normalized.EmailEnabled,
		DigestEnabled:    normalized.DigestEnabled,
		UpdatedAt:        s.clock.Now(),
	})
	if err != nil {
		return AuthorSubscriptionResult{}, mapPortsError(err)
	}
	if err := s.unread.Delete(ctx, authorSubscriptionCacheKey(command.Actor.UserID, command.AuthorID)); err != nil {
		return AuthorSubscriptionResult{}, fmt.Errorf("invalidate notification author subscription cache: %w", err)
	}
	return authorSubscriptionResult(result), nil
}

func notificationPreferencesResult(input ports.NotificationPreferences) NotificationPreferencesResult {
	grouped := make(map[string]NotificationChannelPreferenceInput)
	order := make([]string, 0)
	for _, item := range input.Preferences {
		notificationType := item.NotificationType
		if _, ok := grouped[notificationType]; !ok {
			order = append(order, notificationType)
		}
		channels := grouped[notificationType]
		switch item.Channel {
		case preference.ChannelInApp:
			channels.InApp = item.Enabled
		case preference.ChannelWebsocket:
			channels.Websocket = item.Enabled
		case preference.ChannelEmail:
			channels.Email = item.Enabled
		case preference.ChannelSMS:
			channels.SMS = item.Enabled
		}
		grouped[notificationType] = channels
	}
	result := NotificationPreferencesResult{UserID: input.UserID, Preferences: make([]NotificationPreferenceResult, 0, len(order))}
	for _, notificationType := range order {
		result.Preferences = append(result.Preferences, NotificationPreferenceResult{
			NotificationType: notificationType,
			Channels:         grouped[notificationType],
		})
	}
	return result
}

func notificationDNDResult(input ports.NotificationDND) NotificationDNDResult {
	return NotificationDNDResult{
		UserID:     input.UserID,
		Enabled:    input.Enabled,
		StartTime:  input.StartTime,
		EndTime:    input.EndTime,
		Timezone:   input.Timezone,
		Categories: input.Categories,
		Channels:   input.Channels,
	}
}

func authorSubscriptionResult(input ports.AuthorSubscription) AuthorSubscriptionResult {
	return AuthorSubscriptionResult{
		UserID:           input.UserID,
		AuthorID:         input.AuthorID,
		Level:            input.Level,
		InAppEnabled:     input.InAppEnabled,
		WebsocketEnabled: input.WebsocketEnabled,
		EmailEnabled:     input.EmailEnabled,
		DigestEnabled:    input.DigestEnabled,
	}
}
