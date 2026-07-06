package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
)

func TestPostScheduleReturnsSuccessEnvelope(t *testing.T) {
	scheduledAt := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	service := &fakeContentService{scheduleResult: application.SchedulePostResult{
		PostID:      "post_1",
		PostVersion: 6,
		Status:      "SCHEDULED",
		ScheduledAt: scheduledAt,
	}}
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_1/schedule", bytes.NewBufferString(`{
		"basePostVersion":5,
		"draftBodyId":"body_draft",
		"draftBodyHash":"sha256:draft",
		"scheduledAt":"2026-07-06T12:00:00Z"
	}`))), "42")

	NewHandler(service).ServeHTTP(rr, req)

	if service.scheduleCmd.Actor == nil || service.scheduleCmd.Actor.UserID != 42 ||
		service.scheduleCmd.PostID != "post_1" || service.scheduleCmd.BasePostVersion != 5 ||
		service.scheduleCmd.DraftBodyID != "body_draft" || service.scheduleCmd.DraftBodyHash != "sha256:draft" ||
		!service.scheduleCmd.ScheduledAt.Equal(scheduledAt) {
		t.Fatalf("schedule command = %#v", service.scheduleCmd)
	}
	var body envelope[schedulePostResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Timestamp <= 0 {
		t.Fatalf("status=%d envelope=%#v body=%s", rr.Code, body, rr.Body.String())
	}
	if body.Data.PostID != "post_1" || body.Data.PostVersion != 6 ||
		body.Data.Status != "SCHEDULED" || body.Data.ScheduledAt != "2026-07-06T12:00:00Z" {
		t.Fatalf("data = %#v", body.Data)
	}
}

func TestPostScheduleMapsErrorsAndValidatesRequest(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "invalid body", body: `{"basePostVersion":0,"draftBodyId":"body","draftBodyHash":"hash","scheduledAt":"2026-07-06T12:00:00Z"}`, wantStatus: http.StatusBadRequest, wantCode: 1001},
		{name: "invalid time", body: `{"basePostVersion":1,"draftBodyId":"body","draftBodyHash":"hash","scheduledAt":"not-time"}`, wantStatus: http.StatusBadRequest, wantCode: 1001},
		{name: "not draft conflict", body: validScheduleJSON(), err: domain.ErrDraftConflict, wantStatus: http.StatusConflict, wantCode: 4017},
		{name: "body too short", body: validScheduleJSON(), err: domain.ErrBodyTooShort, wantStatus: http.StatusBadRequest, wantCode: 4016},
		{name: "media invalid", body: validScheduleJSON(), err: application.ErrMediaRefInvalid, wantStatus: http.StatusBadRequest, wantCode: 4021},
		{name: "cover unavailable", body: validScheduleJSON(), err: application.ErrCoverUnavailable, wantStatus: http.StatusBadRequest, wantCode: 4023},
		{name: "dependency unavailable", body: validScheduleJSON(), err: application.ErrDependencyUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 1004},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeContentService{scheduleErr: tt.err}
			rr := httptest.NewRecorder()
			req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_1/schedule", bytes.NewBufferString(tt.body))), "42")

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestPostCancelScheduleReturnsSuccessEnvelope(t *testing.T) {
	service := &fakeContentService{cancelScheduleResult: application.PostLifecycleResult{
		PostID:      "post_1",
		PostVersion: 7,
		Status:      "DRAFT",
		UpdatedAt:   time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC),
	}}
	rr := httptest.NewRecorder()
	req := withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/posts/post_1/schedule?basePostVersion=6", nil), "42")

	NewHandler(service).ServeHTTP(rr, req)

	if service.cancelScheduleCmd.Actor == nil || service.cancelScheduleCmd.Actor.UserID != 42 ||
		service.cancelScheduleCmd.PostID != "post_1" || service.cancelScheduleCmd.BasePostVersion != 6 {
		t.Fatalf("cancel command = %#v", service.cancelScheduleCmd)
	}
	var body envelope[postLifecycleResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Data.Status != "DRAFT" || body.Data.PostVersion != 7 {
		t.Fatalf("status=%d body=%#v raw=%s", rr.Code, body, rr.Body.String())
	}
}

func TestPostCancelScheduleMapsNotScheduled(t *testing.T) {
	service := &fakeContentService{cancelScheduleErr: domain.ErrPostNotPublished}
	rr := httptest.NewRecorder()
	req := withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/posts/post_1/schedule?basePostVersion=6", nil), "42")

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusConflict, 4003)
}

func validScheduleJSON() string {
	return `{"basePostVersion":5,"draftBodyId":"body_draft","draftBodyHash":"sha256:draft","scheduledAt":"2026-07-06T12:00:00Z"}`
}
