package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/application"
)

func TestMarkNotificationReadMapsInvalidPublicIDToBadRequest(t *testing.T) {
	service := &fakeService{markReadErr: application.ErrInvalidRequest}
	router := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/bad-id/read", nil)
	req.Header.Set("X-User-Id", "42")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != 1001 {
		t.Fatalf("code = %d, want 1001", body.Code)
	}
}

func TestMarkAllNotificationsReadAliasRoutesToSameCommand(t *testing.T) {
	service := &fakeService{markAllResult: application.MarkAllNotificationsReadResult{
		ReadAll:       true,
		AffectedCount: 3,
		ReadAt:        time.Date(2026, 7, 6, 16, 0, 0, 0, time.UTC),
	}}
	router := NewHandler(service)

	for _, path := range []string{"/api/v1/notifications/read-all", "/api/v1/notifications/mark-all-read"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("X-User-Id", "42")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200; body=%s", path, rr.Code, rr.Body.String())
		}
	}
	if service.markAllCalls != 2 {
		t.Fatalf("mark all calls = %d, want 2", service.markAllCalls)
	}
}

func TestMarkNotificationGroupReadRoutesPublicGroupID(t *testing.T) {
	router := NewHandler(&fakeService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification-groups/ng1F7qK2m/read", nil)
	req.Header.Set("X-User-Id", "42")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
}

func TestListNotificationsUsesGroupContractShape(t *testing.T) {
	service := &fakeService{listResult: application.NotificationPage{Items: []application.AggregatedNotification{{
		GroupID: "ng1abc", Type: "NEW_FUTURE_TYPE", Category: "INTERACTION", TotalCount: 5, UnreadCount: 2,
		ActorTotalCount: 4,
		RecentActors: []application.NotificationActorSnapshot{
			{PublicID: "user_1", DisplayName: "陈立"},
			{PublicID: "user_2", DisplayName: "小郑"},
			{PublicID: "user_3", DisplayName: "阿宋"},
		},
		LatestTime: time.Date(2026, 7, 10, 5, 30, 0, 0, time.UTC), LatestContent: "陈立赞了你的文章",
	}}}}
	router := NewHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req.Header.Set("X-User-Id", "42")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Data struct {
			Items []struct {
				GroupID         string `json:"groupId"`
				Type            string `json:"type"`
				ActorTotalCount int64  `json:"actorTotalCount"`
				RecentActors    []struct {
					PublicID string `json:"publicId"`
				} `json:"recentActors"`
				Target *json.RawMessage `json:"target"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].GroupID != "ng1abc" || body.Data.Items[0].Type != "NEW_FUTURE_TYPE" || body.Data.Items[0].ActorTotalCount != 4 || len(body.Data.Items[0].RecentActors) != 3 || body.Data.Items[0].RecentActors[0].PublicID != "user_1" || body.Data.Items[0].Target != nil {
		t.Fatalf("data = %#v", body.Data.Items)
	}
}

func TestListNotificationGroupActorsHidesNonOwnerAsNotFound(t *testing.T) {
	router := NewHandler(&fakeService{listGroupActorsErr: application.ErrNotificationNotFound})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-groups/ng1other/actors", nil)
	req.Header.Set("X-User-Id", "42")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestListNotificationsRejectsInvalidContractFilters(t *testing.T) {
	router := NewHandler(&fakeService{})
	for _, rawQuery := range []string{"category=UNKNOWN", "unreadOnly=not-a-bool", "size=51"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?"+rawQuery, nil)
		req.Header.Set("X-User-Id", "42")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("query %q status = %d, want 400; body=%s", rawQuery, rr.Code, rr.Body.String())
		}
	}
}

func TestNotificationPreferenceCanonicalAndAliasRoutes(t *testing.T) {
	service := &fakeService{}
	router := NewHandler(service)

	for _, path := range []string{"/api/v1/notification-preferences", "/api/v1/notifications/preferences"} {
		req := httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(`{
			"preferences":[{"notificationType":"POST_LIKED","channels":{"inApp":true,"websocket":true,"email":false,"sms":false}}]
		}`))
		req.Header.Set("X-User-Id", "42")
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200; body=%s", path, rr.Code, rr.Body.String())
		}
	}
	if service.updatePreferencesCalls != 2 {
		t.Fatalf("update preference calls = %d, want 2", service.updatePreferencesCalls)
	}
	if service.lastPreference.Actor.UserID != 42 || len(service.lastPreference.Preferences) != 1 || !service.lastPreference.Preferences[0].Channels.Websocket {
		t.Fatalf("last preference command = %#v", service.lastPreference)
	}
}

func TestUpdateNotificationDNDRouteMapsPayload(t *testing.T) {
	service := &fakeService{}
	router := NewHandler(service)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/notification-dnd", bytes.NewBufferString(`{
		"enabled":true,
		"startTime":"22:00",
		"endTime":"07:00",
		"timezone":"Asia/Shanghai",
		"categories":["INTERACTION"],
		"channels":["WEBSOCKET","EMAIL"]
	}`))
	req.Header.Set("X-User-Id", "42")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if service.lastDND.Actor.UserID != 42 || service.lastDND.StartTime != "22:00" || service.lastDND.EndTime != "07:00" {
		t.Fatalf("last dnd command = %#v", service.lastDND)
	}
}

func TestRetryDeliveryRouteParsesDeliveryID(t *testing.T) {
	service := &fakeService{}
	router := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification-deliveries/d1retry/retry", nil)
	req.Header.Set("X-User-Id", "42")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if service.lastRetry.Actor.UserID != 42 || service.lastRetry.DeliveryID != "d1retry" {
		t.Fatalf("last retry command = %#v", service.lastRetry)
	}
}

type fakeService struct {
	markReadErr            error
	markAllCalls           int
	markAllResult          application.MarkAllNotificationsReadResult
	updatePreferencesCalls int
	lastPreference         application.UpdateNotificationPreferencesCommand
	lastDND                application.UpdateNotificationDNDCommand
	lastRetry              application.RetryDeliveryCommand
	listResult             application.NotificationPage
	listGroupActorsErr     error
}

func (f *fakeService) MarkNotificationRead(ctx context.Context, cmd application.MarkNotificationReadCommand) (application.MarkNotificationReadResult, error) {
	if f.markReadErr != nil {
		return application.MarkNotificationReadResult{}, f.markReadErr
	}
	return application.MarkNotificationReadResult{NotificationID: cmd.NotificationID, Read: true, ReadAt: time.Now().UTC()}, nil
}

func (f *fakeService) MarkAllNotificationsRead(ctx context.Context, cmd application.MarkAllNotificationsReadCommand) (application.MarkAllNotificationsReadResult, error) {
	f.markAllCalls++
	return f.markAllResult, nil
}

func (f *fakeService) GetUnreadCount(ctx context.Context, query application.GetUnreadCountQuery) (application.UnreadCountResult, error) {
	return application.UnreadCountResult{}, nil
}

func (f *fakeService) GetUnreadBreakdown(ctx context.Context, query application.GetUnreadBreakdownQuery) (application.UnreadBreakdownResult, error) {
	return application.UnreadBreakdownResult{}, nil
}

func (f *fakeService) ListAggregatedNotifications(ctx context.Context, query application.ListNotificationsQuery) (application.NotificationPage, error) {
	return f.listResult, nil
}

func (f *fakeService) ListNotificationGroupActors(ctx context.Context, query application.ListNotificationGroupActorsQuery) (application.NotificationActorPage, error) {
	return application.NotificationActorPage{}, f.listGroupActorsErr
}

func (f *fakeService) MarkNotificationGroupRead(ctx context.Context, cmd application.MarkNotificationGroupReadCommand) (application.MarkNotificationGroupReadResult, error) {
	return application.MarkNotificationGroupReadResult{GroupID: cmd.GroupID, Read: true, ReadAt: time.Now().UTC()}, nil
}

func (f *fakeService) GetNotificationPreferences(ctx context.Context, query application.GetNotificationPreferencesQuery) (application.NotificationPreferencesResult, error) {
	return application.NotificationPreferencesResult{}, nil
}

func (f *fakeService) UpdateNotificationPreferences(ctx context.Context, cmd application.UpdateNotificationPreferencesCommand) (application.NotificationPreferencesResult, error) {
	f.updatePreferencesCalls++
	f.lastPreference = cmd
	preferences := make([]application.NotificationPreferenceResult, 0, len(cmd.Preferences))
	for _, item := range cmd.Preferences {
		preferences = append(preferences, application.NotificationPreferenceResult{NotificationType: item.NotificationType, Channels: item.Channels})
	}
	return application.NotificationPreferencesResult{UserID: cmd.Actor.UserID, Preferences: preferences}, nil
}

func (f *fakeService) GetNotificationDND(ctx context.Context, query application.GetNotificationDNDQuery) (application.NotificationDNDResult, error) {
	return application.NotificationDNDResult{}, nil
}

func (f *fakeService) UpdateNotificationDND(ctx context.Context, cmd application.UpdateNotificationDNDCommand) (application.NotificationDNDResult, error) {
	f.lastDND = cmd
	return application.NotificationDNDResult{
		UserID:     cmd.Actor.UserID,
		Enabled:    cmd.Enabled,
		StartTime:  cmd.StartTime,
		EndTime:    cmd.EndTime,
		Timezone:   cmd.Timezone,
		Categories: cmd.Categories,
		Channels:   cmd.Channels,
	}, nil
}

func (f *fakeService) GetAuthorSubscription(ctx context.Context, query application.GetAuthorSubscriptionQuery) (application.AuthorSubscriptionResult, error) {
	return application.AuthorSubscriptionResult{}, nil
}

func (f *fakeService) UpdateAuthorSubscription(ctx context.Context, cmd application.UpdateAuthorSubscriptionCommand) (application.AuthorSubscriptionResult, error) {
	return application.AuthorSubscriptionResult{}, nil
}

func (f *fakeService) ListDeliveries(ctx context.Context, query application.ListDeliveriesQuery) (application.DeliveryPage, error) {
	return application.DeliveryPage{}, nil
}

func (f *fakeService) RetryDelivery(ctx context.Context, cmd application.RetryDeliveryCommand) (application.DeliveryRetryResult, error) {
	f.lastRetry = cmd
	return application.DeliveryRetryResult{DeliveryID: cmd.DeliveryID, Status: "WEBSOCKET_PENDING", Retried: true}, nil
}

var _ = errors.Is
