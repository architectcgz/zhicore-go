package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/application"
)

func TestCreateCommentRequiresTrustedUserID(t *testing.T) {
	service := &fakeCommentService{}
	rr := httptest.NewRecorder()
	req := withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_pub_1/comments", bytes.NewBufferString(`{"content":"hello"}`)))

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
	if service.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0", service.createCalls)
	}
}

func TestCreateCommentForwardsTrustedActorAndBody(t *testing.T) {
	createdAt := time.Date(2026, 7, 4, 16, 0, 0, 0, time.UTC)
	service := &fakeCommentService{createResult: application.CreateCommentResult{
		PostID:          "post_pub_1",
		CommentID:       "c7001",
		RootCommentID:   "c7000",
		ParentCommentID: "c6999",
		CreatedAt:       createdAt,
	}}
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_pub_1/comments", bytes.NewBufferString(`{"content":" reply ","parentCommentId":"c6999","imageFileIds":["img_1"],"voiceDuration":0,"userId":999}`))), "42")

	NewHandler(service).ServeHTTP(rr, req)

	if service.createCmd.ActorUserID != 42 || service.createCmd.PostID != "post_pub_1" || service.createCmd.ParentCommentID != "c6999" || service.createCmd.Content != " reply " {
		t.Fatalf("create command = %#v", service.createCmd)
	}
	if len(service.createCmd.ImageFileIDs) != 1 || service.createCmd.ImageFileIDs[0] != "img_1" {
		t.Fatalf("image ids = %#v", service.createCmd.ImageFileIDs)
	}
	var body envelope[createCommentResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Data.CommentID != "c7001" || body.Data.RootCommentID != "c7000" || body.Data.ParentCommentID != "c6999" {
		t.Fatalf("status=%d body=%s decoded=%#v", rr.Code, rr.Body.String(), body)
	}
	if body.Data.CreatedAt != createdAt.Format(time.RFC3339) {
		t.Fatalf("createdAt = %q, want %q", body.Data.CreatedAt, createdAt.Format(time.RFC3339))
	}
}

func TestCreateCommentMapsApplicationErrorsToPublicCodes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "empty content", err: application.ErrCommentContentRequired, wantStatus: http.StatusBadRequest, wantCode: 5003},
		{name: "content too long", err: application.ErrCommentContentTooLong, wantStatus: http.StatusBadRequest, wantCode: 5004},
		{name: "post not found", err: application.ErrPostNotFound, wantStatus: http.StatusNotFound, wantCode: 4001},
		{name: "parent not found", err: application.ErrParentCommentNotFound, wantStatus: http.StatusNotFound, wantCode: 5006},
		{name: "dependency unavailable", err: application.ErrDependencyUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 1004},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeCommentService{createErr: tc.err}
			rr := httptest.NewRecorder()
			req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_pub_1/comments", bytes.NewBufferString(`{"content":"hello"}`))), "42")

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}
