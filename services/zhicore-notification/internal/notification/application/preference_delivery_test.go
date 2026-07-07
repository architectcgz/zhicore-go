package application

import (
	"context"
	"errors"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func TestUpdateNotificationPreferencesRejectsEnabledSMS(t *testing.T) {
	deps := newPreferenceDeliveryDeps()
	service := mustNewService(t, deps.dependencies())

	_, err := service.UpdateNotificationPreferences(context.Background(), UpdateNotificationPreferencesCommand{
		Actor: Actor{UserID: 42},
		Preferences: []NotificationPreferenceInput{{
			NotificationType: "POST_LIKED",
			Channels: NotificationChannelPreferenceInput{
				InApp: true,
				SMS:   true,
			},
		}},
	})

	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("UpdateNotificationPreferences() error = %v, want %v", err, ErrInvalidRequest)
	}
	if deps.settings.savePreferencesCalls != 0 {
		t.Fatalf("save preferences calls = %d, want 0", deps.settings.savePreferencesCalls)
	}
}

func TestUpdateNotificationDNDAllowsCrossDayWindowButRejectsEqualTimes(t *testing.T) {
	deps := newPreferenceDeliveryDeps()
	service := mustNewService(t, deps.dependencies())

	result, err := service.UpdateNotificationDND(context.Background(), UpdateNotificationDNDCommand{
		Actor:      Actor{UserID: 42},
		Enabled:    true,
		StartTime:  "22:00",
		EndTime:    "07:00",
		Timezone:   "Asia/Shanghai",
		Categories: []string{"INTERACTION", "SOCIAL"},
		Channels:   []string{"WEBSOCKET", "EMAIL"},
	})
	if err != nil {
		t.Fatalf("UpdateNotificationDND() error = %v", err)
	}
	if !result.Enabled || result.StartTime != "22:00" || result.EndTime != "07:00" || result.Timezone != "Asia/Shanghai" {
		t.Fatalf("dnd result = %#v", result)
	}
	if deps.settings.lastDND.UserID != 42 || deps.settings.lastDND.StartTime != "22:00" || deps.settings.lastDND.EndTime != "07:00" {
		t.Fatalf("saved dnd = %#v", deps.settings.lastDND)
	}

	_, err = service.UpdateNotificationDND(context.Background(), UpdateNotificationDNDCommand{
		Actor:     Actor{UserID: 42},
		Enabled:   true,
		StartTime: "08:00",
		EndTime:   "08:00",
		Timezone:  "UTC",
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("equal DND times error = %v, want %v", err, ErrInvalidRequest)
	}
}

func TestUpdateAuthorSubscriptionNormalizesDigestOnlyAndMutedLevels(t *testing.T) {
	deps := newPreferenceDeliveryDeps()
	service := mustNewService(t, deps.dependencies())

	result, err := service.UpdateAuthorSubscription(context.Background(), UpdateAuthorSubscriptionCommand{
		Actor:            Actor{UserID: 42},
		AuthorID:         1001,
		Level:            "DIGEST_ONLY",
		InAppEnabled:     true,
		WebsocketEnabled: true,
		EmailEnabled:     true,
		DigestEnabled:    false,
	})
	if err != nil {
		t.Fatalf("UpdateAuthorSubscription(DIGEST_ONLY) error = %v", err)
	}
	if result.InAppEnabled || result.WebsocketEnabled || result.EmailEnabled || !result.DigestEnabled {
		t.Fatalf("digest-only result = %#v, want only digest enabled", result)
	}

	result, err = service.UpdateAuthorSubscription(context.Background(), UpdateAuthorSubscriptionCommand{
		Actor:            Actor{UserID: 42},
		AuthorID:         1001,
		Level:            "MUTED",
		InAppEnabled:     true,
		WebsocketEnabled: true,
		EmailEnabled:     true,
		DigestEnabled:    true,
	})
	if err != nil {
		t.Fatalf("UpdateAuthorSubscription(MUTED) error = %v", err)
	}
	if result.InAppEnabled || result.WebsocketEnabled || result.EmailEnabled || result.DigestEnabled {
		t.Fatalf("muted result = %#v, want all channels disabled", result)
	}
}

type preferenceDeliveryDeps struct {
	readTestDeps
	settings *fakeSettingsRepository
}

func newPreferenceDeliveryDeps() preferenceDeliveryDeps {
	readDeps := newReadTestDeps()
	return preferenceDeliveryDeps{
		readTestDeps: readDeps,
		settings:     &fakeSettingsRepository{},
	}
}

func (d preferenceDeliveryDeps) dependencies() Dependencies {
	deps := d.readTestDeps.dependencies()
	deps.Settings = d.settings
	return deps
}

type fakeSettingsRepository struct {
	savePreferencesCalls int
	lastPreferences      ports.SaveNotificationPreferencesInput
	lastDND              ports.SaveNotificationDNDInput
	lastSubscription     ports.SaveAuthorSubscriptionInput
}

func (f *fakeSettingsRepository) GetNotificationPreferences(ctx context.Context, userID int64) (ports.NotificationPreferences, error) {
	return ports.NotificationPreferences{}, nil
}

func (f *fakeSettingsRepository) SaveNotificationPreferences(ctx context.Context, input ports.SaveNotificationPreferencesInput) (ports.NotificationPreferences, error) {
	f.savePreferencesCalls++
	f.lastPreferences = input
	return ports.NotificationPreferences{UserID: input.UserID, Preferences: input.Preferences}, nil
}

func (f *fakeSettingsRepository) GetNotificationDND(ctx context.Context, userID int64) (ports.NotificationDND, error) {
	return ports.NotificationDND{}, nil
}

func (f *fakeSettingsRepository) SaveNotificationDND(ctx context.Context, input ports.SaveNotificationDNDInput) (ports.NotificationDND, error) {
	f.lastDND = input
	return ports.NotificationDND{
		UserID:     input.UserID,
		Enabled:    input.Enabled,
		StartTime:  input.StartTime,
		EndTime:    input.EndTime,
		Timezone:   input.Timezone,
		Categories: input.Categories,
		Channels:   input.Channels,
	}, nil
}

func (f *fakeSettingsRepository) GetAuthorSubscription(ctx context.Context, input ports.GetAuthorSubscriptionInput) (ports.AuthorSubscription, error) {
	return ports.AuthorSubscription{}, nil
}

func (f *fakeSettingsRepository) SaveAuthorSubscription(ctx context.Context, input ports.SaveAuthorSubscriptionInput) (ports.AuthorSubscription, error) {
	f.lastSubscription = input
	return ports.AuthorSubscription{
		UserID:           input.UserID,
		AuthorID:         input.AuthorID,
		Level:            input.Level,
		InAppEnabled:     input.InAppEnabled,
		WebsocketEnabled: input.WebsocketEnabled,
		EmailEnabled:     input.EmailEnabled,
		DigestEnabled:    input.DigestEnabled,
	}, nil
}
