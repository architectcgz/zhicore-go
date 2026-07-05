package httpapi

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
)

func TestAdminOutboxRequiresAdminRole(t *testing.T) {
	service := &fakeContentService{}
	rr := httptest.NewRecorder()
	req := withUserID(httptest.NewRequest(http.MethodGet, "/api/v1/admin/content/outbox-events?status=failed", nil), "42")

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusForbidden, 2007)
	if service.listOutboxCalls != 0 {
		t.Fatalf("listOutboxCalls = %d, want 0", service.listOutboxCalls)
	}
}

func TestAdminOutboxRequiresLoginBeforeAdminRole(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   *bytes.Buffer
	}{
		{name: "list", method: http.MethodGet, path: "/api/v1/admin/content/outbox-events?status=failed"},
		{name: "retry", method: http.MethodPost, path: "/api/v1/admin/content/outbox-events/evt_1/retry", body: bytes.NewBufferString(`{"reason":"manual replay"}`)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeContentService{}
			rr := httptest.NewRecorder()
			var body io.Reader
			if tc.body != nil {
				body = bytes.NewBuffer(tc.body.Bytes())
			}
			req := withRoles(httptest.NewRequest(tc.method, tc.path, body), "admin")
			if tc.method == http.MethodPost {
				req = withJSON(req)
			}

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
			if service.listOutboxCalls != 0 || service.retryOutboxCalls != 0 {
				t.Fatalf("service calls list=%d retry=%d, want none", service.listOutboxCalls, service.retryOutboxCalls)
			}
		})
	}
}

func TestAdminOutboxListMapsQueryAndResponse(t *testing.T) {
	occurredAt := time.Date(2026, 7, 5, 15, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 7, 5, 15, 1, 0, 0, time.UTC)
	service := &fakeContentService{listOutboxResult: application.ListAdminOutboxEventsResult{
		Items: []application.AdminOutboxEventItem{{
			EventID:          "evt_post_published_1",
			EventType:        "content.post.published",
			AggregateType:    "post",
			AggregateID:      "post_1",
			AggregateVersion: 6,
			Status:           "FAILED",
			RetryCount:       2,
			LastError:        "rabbitmq publish failed",
			OccurredAt:       occurredAt,
			CreatedAt:        occurredAt,
			UpdatedAt:        updatedAt,
		}},
		Page:  2,
		Size:  20,
		Total: 21,
	}}
	rr := httptest.NewRecorder()
	req := withRoles(withUserID(httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/content/outbox-events?status=dead&eventType=content.post.published&page=2&size=20",
		nil,
	), "42"), "writer,admin")

	NewHandler(service).ServeHTTP(rr, req)

	if service.listOutboxCalls != 1 {
		t.Fatalf("listOutboxCalls = %d, want 1", service.listOutboxCalls)
	}
	if service.listOutboxQuery.Actor == nil || service.listOutboxQuery.Actor.UserID != 42 || !service.listOutboxQuery.Actor.HasRole("admin") {
		t.Fatalf("actor = %#v, want admin 42", service.listOutboxQuery.Actor)
	}
	if service.listOutboxQuery.Status != "dead" || service.listOutboxQuery.EventType != "content.post.published" || service.listOutboxQuery.Page != 2 || service.listOutboxQuery.Size != 20 {
		t.Fatalf("query = %+v", service.listOutboxQuery)
	}

	var body envelope[adminOutboxListResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Data.Total != 21 || len(body.Data.Items) != 1 {
		t.Fatalf("status=%d body=%#v raw=%s", rr.Code, body, rr.Body.String())
	}
	item := body.Data.Items[0]
	if item.EventID != "evt_post_published_1" || item.AggregateVersion != 6 || item.RetryCount != 2 || item.UpdatedAt != formatTime(updatedAt) {
		t.Fatalf("item = %+v", item)
	}
}

func TestAdminOutboxRetryMapsReasonAndResponse(t *testing.T) {
	retriedAt := time.Date(2026, 7, 5, 15, 2, 0, 0, time.UTC)
	service := &fakeContentService{retryOutboxResult: application.RetryAdminOutboxEventResult{
		EventID:    "evt_post_published_1",
		Status:     "PENDING",
		RetryCount: 2,
		RetriedAt:  retriedAt,
	}}
	rr := httptest.NewRecorder()
	req := withRoles(withUserID(withJSON(httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/content/outbox-events/evt_post_published_1/retry",
		bytes.NewBufferString(`{"reason":"manual replay after RabbitMQ recovery"}`),
	)), "42"), "ROLE_ADMIN")

	NewHandler(service).ServeHTTP(rr, req)

	if service.retryOutboxCalls != 1 {
		t.Fatalf("retryOutboxCalls = %d, want 1", service.retryOutboxCalls)
	}
	if service.retryOutboxCommand.EventID != "evt_post_published_1" || service.retryOutboxCommand.Reason != "manual replay after RabbitMQ recovery" {
		t.Fatalf("retry command = %+v", service.retryOutboxCommand)
	}
	if service.retryOutboxCommand.Actor == nil || service.retryOutboxCommand.Actor.UserID != 42 || !service.retryOutboxCommand.Actor.HasRole("role_admin") {
		t.Fatalf("actor = %#v, want role_admin 42", service.retryOutboxCommand.Actor)
	}

	var body envelope[adminOutboxRetryResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Data.EventID != "evt_post_published_1" || body.Data.Status != "PENDING" || body.Data.RetriedAt != formatTime(retriedAt) {
		t.Fatalf("status=%d body=%#v raw=%s", rr.Code, body, rr.Body.String())
	}
}

func TestAdminOutboxRetryRequiresReason(t *testing.T) {
	service := &fakeContentService{}
	rr := httptest.NewRecorder()
	req := withRoles(withUserID(withJSON(httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/content/outbox-events/evt_post_published_1/retry",
		bytes.NewBufferString(`{"reason":" "}`),
	)), "42"), "admin")

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusBadRequest, 1001)
	if service.retryOutboxCalls != 0 {
		t.Fatalf("retryOutboxCalls = %d, want 0", service.retryOutboxCalls)
	}
}
