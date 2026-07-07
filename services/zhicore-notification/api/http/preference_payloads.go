package httpapi

import (
	"github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/application"
)

func notificationPreferenceInputs(input []notificationPreferencePayload) []application.NotificationPreferenceInput {
	out := make([]application.NotificationPreferenceInput, 0, len(input))
	for _, item := range input {
		out = append(out, application.NotificationPreferenceInput{
			NotificationType: item.NotificationType,
			Channels: application.NotificationChannelPreferenceInput{
				InApp:     item.Channels.InApp,
				Websocket: item.Channels.Websocket,
				Email:     item.Channels.Email,
				SMS:       item.Channels.SMS,
			},
		})
	}
	return out
}

func notificationPreferencesResponse(input application.NotificationPreferencesResult) notificationPreferencesResp {
	items := make([]notificationPreferencePayload, 0, len(input.Preferences))
	for _, item := range input.Preferences {
		items = append(items, notificationPreferencePayload{
			NotificationType: item.NotificationType,
			Channels: notificationChannelPreferencePayload{
				InApp:     item.Channels.InApp,
				Websocket: item.Channels.Websocket,
				Email:     item.Channels.Email,
				SMS:       item.Channels.SMS,
			},
		})
	}
	return notificationPreferencesResp{UserID: input.UserID, Preferences: items}
}

func notificationDNDResponse(input application.NotificationDNDResult) notificationDNDResp {
	return notificationDNDResp{
		UserID:     input.UserID,
		Enabled:    input.Enabled,
		StartTime:  input.StartTime,
		EndTime:    input.EndTime,
		Timezone:   input.Timezone,
		Categories: input.Categories,
		Channels:   input.Channels,
	}
}

func authorSubscriptionResponse(input application.AuthorSubscriptionResult) authorSubscriptionResp {
	return authorSubscriptionResp{
		UserID:           input.UserID,
		AuthorID:         input.AuthorID,
		Level:            input.Level,
		InAppEnabled:     input.InAppEnabled,
		WebsocketEnabled: input.WebsocketEnabled,
		EmailEnabled:     input.EmailEnabled,
		DigestEnabled:    input.DigestEnabled,
	}
}

func deliveryPageResponse(input application.DeliveryPage) deliveryPageResp {
	items := make([]deliveryResp, 0, len(input.Items))
	for _, item := range input.Items {
		var nextRetryAt *string
		if item.NextRetryAt != nil {
			formatted := httpapi.FormatRFC3339UTC(*item.NextRetryAt)
			nextRetryAt = &formatted
		}
		items = append(items, deliveryResp{
			DeliveryID:       item.DeliveryID,
			RecipientID:      item.RecipientID,
			NotificationID:   item.NotificationID,
			Channel:          item.Channel,
			NotificationType: item.NotificationType,
			Status:           item.Status,
			Provider:         item.Provider,
			AttemptCount:     item.AttemptCount,
			LastErrorCode:    item.LastErrorCode,
			NextRetryAt:      nextRetryAt,
			CreatedAt:        httpapi.FormatRFC3339UTC(item.CreatedAt),
			UpdatedAt:        httpapi.FormatRFC3339UTC(item.UpdatedAt),
		})
	}
	return deliveryPageResp{Items: items, NextCursor: input.NextCursor, HasMore: input.HasMore}
}
