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

func TestPublishPostRequiresLoginAndMapsForbidden(t *testing.T) {
	t.Run("missing trusted user", func(t *testing.T) {
		service := &fakeContentService{}
		rr := httptest.NewRecorder()
		req := withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_pub_1/publish", bytes.NewBufferString(validPublishPostJSON())))

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
		if service.publishCalls != 0 {
			t.Fatalf("publishCalls = %d, want 0", service.publishCalls)
		}
	})

	t.Run("non owner", func(t *testing.T) {
		service := &fakeContentService{publishErr: domain.ErrForbidden}
		rr := httptest.NewRecorder()
		req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_pub_1/publish", bytes.NewBufferString(validPublishPostJSON()))), "42")

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusForbidden, 2008)
	})
}

func TestPublishPostMapsBusinessErrors(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "empty title", err: domain.ErrTitleRequired, wantStatus: http.StatusBadRequest, wantCode: 4005},
		{name: "empty body", err: domain.ErrBodyRequired, wantStatus: http.StatusBadRequest, wantCode: 4006},
		{name: "draft conflict", err: domain.ErrDraftConflict, wantStatus: http.StatusConflict, wantCode: 4017},
		{name: "duplicate publish", err: domain.ErrPostAlreadyPublished, wantStatus: http.StatusConflict, wantCode: 4002},
		{name: "media reference invalid", err: application.ErrMediaRefInvalid, wantStatus: http.StatusBadRequest, wantCode: 4021},
		{name: "cover unavailable", err: application.ErrCoverUnavailable, wantStatus: http.StatusBadRequest, wantCode: 4023},
		{name: "dependency unavailable", err: application.ErrDependencyUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 1004},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeContentService{publishErr: tc.err}
			rr := httptest.NewRecorder()
			req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_pub_1/publish", bytes.NewBufferString(validPublishPostJSON()))), "42")

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestPublishPostReturnsSuccessEnvelope(t *testing.T) {
	publishedAt := time.Date(2026, 7, 5, 9, 0, 0, 0, time.UTC)
	service := &fakeContentService{publishResult: application.PublishPostResult{
		PostID:      "post_pub_1",
		PostVersion: 4,
		PublishedAt: publishedAt,
	}}
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_pub_1/publish", bytes.NewBufferString(validPublishPostJSON()))), "42")

	NewHandler(service).ServeHTTP(rr, req)

	if service.publishCmd.Actor == nil || service.publishCmd.Actor.UserID != 42 || service.publishCmd.PostID != "post_pub_1" {
		t.Fatalf("publish command = %#v", service.publishCmd)
	}
	if service.publishCmd.BasePostVersion != 3 || service.publishCmd.DraftBodyID != "body_draft_2" || service.publishCmd.DraftBodyHash != "sha256:abc" {
		t.Fatalf("publish base fields = %#v", service.publishCmd)
	}
	var body envelope[publishPostResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Timestamp <= 0 {
		t.Fatalf("status=%d envelope=%#v body=%s", rr.Code, body, rr.Body.String())
	}
	if body.Data.PostID != "post_pub_1" || body.Data.PostVersion != 4 || body.Data.PublishedAt != "2026-07-05T09:00:00Z" {
		t.Fatalf("data = %#v", body.Data)
	}
}

func validPublishPostJSON() string {
	return `{"basePostVersion":3,"draftBodyId":"body_draft_2","draftBodyHash":"sha256:abc"}`
}
