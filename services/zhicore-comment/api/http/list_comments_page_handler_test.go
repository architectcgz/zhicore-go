package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/application"
)

func TestListCommentsPageDefaultsQueryAndOmitsAnonymousViewer(t *testing.T) {
	now := time.Date(2026, 7, 4, 17, 0, 0, 0, time.UTC)
	service := &fakeCommentService{pageResult: application.TopLevelCommentPage{
		Items: []application.CommentItem{{
			PostID:    "post_pub_2",
			CommentID: "c8001",
			Author:    application.AuthorSummary{PublicID: "user_pub_1", DisplayName: "Alice"},
			Content:   "hello",
			Status:    application.CommentStatusNormal,
			Stats:     application.CommentStats{LikeCount: 3, ReplyCount: 2},
			CreatedAt: now,
			UpdatedAt: now,
		}},
		Page: 1, Size: 20, TotalComments: 3, TotalTopLevelComments: 1, Pages: 1,
	}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_pub_2/comments/page", nil)

	NewHandler(service).ServeHTTP(rr, req)

	if service.listQuery.PostID != "post_pub_2" || service.listQuery.Page != 0 || service.listQuery.Size != 0 || service.listQuery.Sort != "" || service.listQuery.ViewerUserID != 0 {
		t.Fatalf("list query = %#v", service.listQuery)
	}
	var body envelope[topLevelCommentPageResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Data.TotalComments != 3 || body.Data.Pages != 1 {
		t.Fatalf("status=%d body=%s decoded=%#v", rr.Code, rr.Body.String(), body)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].Viewer != nil {
		t.Fatalf("items = %#v", body.Data.Items)
	}
}

func TestListCommentsPageParsesViewerAndQuery(t *testing.T) {
	service := &fakeCommentService{pageResult: application.TopLevelCommentPage{Page: 2, Size: 10}}
	rr := httptest.NewRecorder()
	req := withUserID(httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_pub_2/comments/page?page=2&size=10&sort=HOT", nil), "88")

	NewHandler(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if service.listQuery.ViewerUserID != 88 || service.listQuery.Page != 2 || service.listQuery.Size != 10 || service.listQuery.Sort != application.CommentSortHot {
		t.Fatalf("list query = %#v", service.listQuery)
	}
}

func TestListCommentsPageRejectsInvalidQuery(t *testing.T) {
	for _, rawURL := range []string{
		"/api/v1/posts/post_pub_2/comments/page?page=0",
		"/api/v1/posts/post_pub_2/comments/page?size=101",
		"/api/v1/posts/post_pub_2/comments/page?sort=BAD",
	} {
		t.Run(rawURL, func(t *testing.T) {
			service := &fakeCommentService{}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, rawURL, nil)

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, http.StatusBadRequest, 1001)
			if service.listCalls != 0 {
				t.Fatalf("listCalls = %d, want 0", service.listCalls)
			}
		})
	}
}

type fakeCommentService struct {
	createCalls  int
	createCmd    application.CreateCommentCommand
	createResult application.CreateCommentResult
	createErr    error
	listCalls    int
	listQuery    application.ListTopLevelCommentsQuery
	pageResult   application.TopLevelCommentPage
	listErr      error
}

func (f *fakeCommentService) CreateComment(ctx context.Context, cmd application.CreateCommentCommand) (application.CreateCommentResult, error) {
	f.createCalls++
	f.createCmd = cmd
	return f.createResult, f.createErr
}

func (f *fakeCommentService) ListTopLevelCommentsByPage(ctx context.Context, query application.ListTopLevelCommentsQuery) (application.TopLevelCommentPage, error) {
	f.listCalls++
	f.listQuery = query
	return f.pageResult, f.listErr
}

type envelope[T any] struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      T      `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

func withJSON(req *http.Request) *http.Request {
	req.Header.Set("Content-Type", "application/json")
	return req
}

func withUserID(req *http.Request, userID string) *http.Request {
	req.Header.Set("X-User-Id", userID)
	return req
}

func decodeJSON(t *testing.T, data []byte, out any) {
	t.Helper()
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", string(data), err)
	}
}

func assertErrorEnvelope(t *testing.T, rr *httptest.ResponseRecorder, wantStatus, wantCode int) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, wantStatus, rr.Body.String())
	}
	var body struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		Timestamp int64  `json:"timestamp"`
	}
	decodeJSON(t, rr.Body.Bytes(), &body)
	if body.Code != wantCode || body.Timestamp <= 0 || body.Message == "" {
		t.Fatalf("error envelope = %#v, want code %d", body, wantCode)
	}
}
