package httpapi

import (
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

type fakeService struct {
	markReadErr   error
	markAllCalls  int
	markAllResult application.MarkAllNotificationsReadResult
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
	return application.NotificationPage{}, nil
}

var _ = errors.Is
