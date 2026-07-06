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

func TestPostLifecycleRequiresTrustedUser(t *testing.T) {
	service := &fakeContentService{}
	rr := httptest.NewRecorder()
	req := withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_1/unpublish", bytes.NewBufferString(`{"basePostVersion":5}`)))

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
	if service.unpublishCalls != 0 {
		t.Fatalf("unpublishCalls = %d, want 0", service.unpublishCalls)
	}
}

func TestPostLifecycleMapsBusinessErrors(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		configure  func(*fakeContentService)
		wantStatus int
		wantCode   int
	}{
		{
			name:   "unpublish forbidden",
			method: http.MethodPost,
			path:   "/api/v1/posts/post_1/unpublish",
			body:   `{"basePostVersion":5}`,
			configure: func(service *fakeContentService) {
				service.unpublishErr = domain.ErrForbidden
			},
			wantStatus: http.StatusForbidden,
			wantCode:   2008,
		},
		{
			name:   "unpublish not published",
			method: http.MethodPost,
			path:   "/api/v1/posts/post_1/unpublish",
			body:   `{"basePostVersion":5}`,
			configure: func(service *fakeContentService) {
				service.unpublishErr = domain.ErrPostNotPublished
			},
			wantStatus: http.StatusConflict,
			wantCode:   4003,
		},
		{
			name:   "delete already deleted",
			method: http.MethodDelete,
			path:   "/api/v1/posts/post_1?basePostVersion=5",
			configure: func(service *fakeContentService) {
				service.deletePostErr = domain.ErrPostDeleted
			},
			wantStatus: http.StatusConflict,
			wantCode:   4004,
		},
		{
			name:   "restore missing or not deleted",
			method: http.MethodPost,
			path:   "/api/v1/posts/post_1/restore",
			body:   `{"basePostVersion":5}`,
			configure: func(service *fakeContentService) {
				service.restoreErr = domain.ErrPostNotFound
			},
			wantStatus: http.StatusNotFound,
			wantCode:   4001,
		},
		{
			name:   "version conflict",
			method: http.MethodPost,
			path:   "/api/v1/posts/post_1/unpublish",
			body:   `{"basePostVersion":5}`,
			configure: func(service *fakeContentService) {
				service.unpublishErr = domain.ErrDraftConflict
			},
			wantStatus: http.StatusConflict,
			wantCode:   4017,
		},
		{
			name:   "dependency unavailable",
			method: http.MethodPost,
			path:   "/api/v1/posts/post_1/unpublish",
			body:   `{"basePostVersion":5}`,
			configure: func(service *fakeContentService) {
				service.unpublishErr = application.ErrDependencyUnavailable
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   1004,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeContentService{}
			tt.configure(service)
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			if tt.body != "" {
				req = withJSON(req)
			}
			req = withUserID(req, "42")

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestPostLifecycleReturnsSuccessEnvelope(t *testing.T) {
	updatedAt := time.Date(2026, 7, 6, 9, 30, 0, 0, time.UTC)
	service := &fakeContentService{unpublishResult: application.PostLifecycleResult{
		PostID:      "post_1",
		PostVersion: 6,
		Status:      "DRAFT",
		UpdatedAt:   updatedAt,
	}}
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_1/unpublish", bytes.NewBufferString(`{"basePostVersion":5}`))), "42")

	NewHandler(service).ServeHTTP(rr, req)

	if service.unpublishCmd.Actor == nil || service.unpublishCmd.Actor.UserID != 42 ||
		service.unpublishCmd.PostID != "post_1" || service.unpublishCmd.BasePostVersion != 5 {
		t.Fatalf("unpublish command = %#v", service.unpublishCmd)
	}
	var body envelope[postLifecycleResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Timestamp <= 0 {
		t.Fatalf("status=%d envelope=%#v body=%s", rr.Code, body, rr.Body.String())
	}
	if body.Data.PostID != "post_1" || body.Data.PostVersion != 6 ||
		body.Data.Status != "DRAFT" || body.Data.UpdatedAt != "2026-07-06T09:30:00Z" {
		t.Fatalf("data = %#v", body.Data)
	}
}

func TestPostLifecycleDeleteAndRestoreForwardVersion(t *testing.T) {
	service := &fakeContentService{
		deletePostResult: application.PostLifecycleResult{PostID: "post_1", PostVersion: 6, Status: "DELETED", UpdatedAt: time.Date(2026, 7, 6, 9, 0, 0, 0, time.UTC)},
		restoreResult:    application.PostLifecycleResult{PostID: "post_1", PostVersion: 7, Status: "DRAFT", UpdatedAt: time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)},
	}

	deleteReq := withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/posts/post_1?basePostVersion=5", nil), "42")
	deleteRR := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(deleteRR, deleteReq)
	assertSuccessCode(t, deleteRR)
	if service.deletePostCmd.BasePostVersion != 5 || service.deletePostCmd.Actor.UserID != 42 {
		t.Fatalf("delete command = %#v", service.deletePostCmd)
	}

	restoreReq := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_1/restore", bytes.NewBufferString(`{"basePostVersion":6}`))), "42")
	restoreRR := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(restoreRR, restoreReq)
	assertSuccessCode(t, restoreRR)
	if service.restoreCmd.BasePostVersion != 6 || service.restoreCmd.Actor.UserID != 42 {
		t.Fatalf("restore command = %#v", service.restoreCmd)
	}
}
