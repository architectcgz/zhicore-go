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

func TestAuthorWorkbenchHandlers(t *testing.T) {
	updatedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	t.Run("lists my posts with trusted actor and query", func(t *testing.T) {
		service := &fakeContentService{listAuthorPostsResult: application.AuthorPostPageResult{
			Items: []application.PostSummary{{PostID: "post_1", AuthorID: "42", Title: "Draft", Status: "DRAFT", CreatedAt: updatedAt, UpdatedAt: updatedAt}},
			Limit: 10,
		}}
		rr := httptest.NewRecorder()
		req := withUserID(httptest.NewRequest(http.MethodGet, "/api/v1/me/posts?status=draft&cursor=cur&limit=10", nil), "42")

		NewHandler(service).ServeHTTP(rr, req)

		if service.listAuthorPostsCalls != 1 || service.listAuthorPostsQuery.Actor.UserID != 42 ||
			service.listAuthorPostsQuery.Status != "draft" || service.listAuthorPostsQuery.Cursor != "cur" || service.listAuthorPostsQuery.Limit != 10 {
			t.Fatalf("query = %+v", service.listAuthorPostsQuery)
		}
		var body envelope[cursorPageResp[postSummaryResp]]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || body.Code != 200 || len(body.Data.Items) != 1 || body.Data.Items[0].PostID != "post_1" {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("requires login for my drafts", func(t *testing.T) {
		service := &fakeContentService{}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me/drafts", nil)

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
		if service.listAuthorDraftsCalls != 0 {
			t.Fatalf("draft calls = %d, want none", service.listAuthorDraftsCalls)
		}
	})

	t.Run("gets own draft with body", func(t *testing.T) {
		service := &fakeContentService{getAuthorDraftResult: application.AuthorDraftResult{
			PostID:      "post_1",
			PostVersion: 5,
			Title:       "Draft",
			Status:      "DRAFT",
			Body: &application.PostBodyResult{
				BodyID:        "body_1",
				SchemaVersion: 1,
				CanonicalJSON: []byte(`{"schemaVersion":1,"blocks":[]}`),
				PlainText:     "draft",
				ContentHash:   "sha256:body",
				SizeBytes:     36,
				CreatedAt:     updatedAt,
			},
			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
		}}
		rr := httptest.NewRecorder()
		req := withUserID(httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_1/draft", nil), "42")

		NewHandler(service).ServeHTTP(rr, req)

		if service.getAuthorDraftCalls != 1 || service.getAuthorDraftQuery.PostID != "post_1" || service.getAuthorDraftQuery.Actor.UserID != 42 {
			t.Fatalf("draft query = %+v", service.getAuthorDraftQuery)
		}
		var body envelope[authorDraftResp]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || body.Data.PostID != "post_1" || body.Data.Body == nil || body.Data.Body.BodyID != "body_1" {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("updates draft meta without trusting body actor", func(t *testing.T) {
		service := &fakeContentService{updateDraftMetaResult: application.DraftMutationResult{PostID: "post_1", PostVersion: 6, Title: "Next"}}
		rr := httptest.NewRecorder()
		req := withUserID(withJSON(httptest.NewRequest(http.MethodPatch, "/api/v1/posts/post_1/draft/meta", bytes.NewBufferString(`{
			"actor":{"userId":999},
			"basePostVersion":5,
			"title":"Next",
			"coverFileId":"file_cover"
		}`))), "42")

		NewHandler(service).ServeHTTP(rr, req)

		if service.updateDraftMetaCalls != 1 || service.updateDraftMetaCommand.Actor.UserID != 42 || service.updateDraftMetaCommand.BasePostVersion != 5 {
			t.Fatalf("command = %+v", service.updateDraftMetaCommand)
		}
		if service.updateDraftMetaCommand.Title == nil || *service.updateDraftMetaCommand.Title != "Next" {
			t.Fatalf("title = %#v", service.updateDraftMetaCommand.Title)
		}
		assertSuccessCode(t, rr)
	})

	t.Run("deletes draft", func(t *testing.T) {
		service := &fakeContentService{deleteDraftResult: application.DraftMutationResult{PostID: "post_1", PostVersion: 6}}
		rr := httptest.NewRecorder()
		req := withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/posts/post_1/draft", nil), "42")

		NewHandler(service).ServeHTTP(rr, req)

		if service.deleteDraftCalls != 1 || service.deleteDraftCommand.PostID != "post_1" || service.deleteDraftCommand.Actor.UserID != 42 {
			t.Fatalf("delete command = %+v", service.deleteDraftCommand)
		}
		assertSuccessCode(t, rr)
	})

	t.Run("maps author workbench errors", func(t *testing.T) {
		cases := []struct {
			name       string
			method     string
			path       string
			body       string
			configure  func(*fakeContentService)
			wantStatus int
			wantCode   int
		}{
			{
				name:       "invalid list limit",
				method:     http.MethodGet,
				path:       "/api/v1/me/posts?limit=0",
				wantStatus: http.StatusBadRequest,
				wantCode:   1001,
			},
			{
				name:   "draft not found",
				method: http.MethodGet,
				path:   "/api/v1/posts/post_missing/draft",
				configure: func(service *fakeContentService) {
					service.getAuthorDraftErr = domain.ErrPostNotFound
				},
				wantStatus: http.StatusNotFound,
				wantCode:   4001,
			},
			{
				name:   "draft forbidden",
				method: http.MethodGet,
				path:   "/api/v1/posts/post_1/draft",
				configure: func(service *fakeContentService) {
					service.getAuthorDraftErr = domain.ErrForbidden
				},
				wantStatus: http.StatusForbidden,
				wantCode:   2008,
			},
			{
				name:   "draft body unavailable",
				method: http.MethodGet,
				path:   "/api/v1/posts/post_1/draft",
				configure: func(service *fakeContentService) {
					service.getAuthorDraftErr = domain.ErrBodyUnavailable
				},
				wantStatus: http.StatusInternalServerError,
				wantCode:   4018,
			},
			{
				name:   "draft body inconsistent",
				method: http.MethodGet,
				path:   "/api/v1/posts/post_1/draft",
				configure: func(service *fakeContentService) {
					service.getAuthorDraftErr = domain.ErrBodyInconsistent
				},
				wantStatus: http.StatusConflict,
				wantCode:   4019,
			},
			{
				name:   "deleted draft",
				method: http.MethodDelete,
				path:   "/api/v1/posts/post_1/draft",
				configure: func(service *fakeContentService) {
					service.deleteDraftErr = domain.ErrPostDeleted
				},
				wantStatus: http.StatusConflict,
				wantCode:   4004,
			},
			{
				name:   "stale draft meta",
				method: http.MethodPatch,
				path:   "/api/v1/posts/post_1/draft/meta",
				body:   `{"basePostVersion":5,"title":"Next"}`,
				configure: func(service *fakeContentService) {
					service.updateDraftMetaErr = domain.ErrDraftConflict
				},
				wantStatus: http.StatusConflict,
				wantCode:   4017,
			},
			{
				name:   "service unavailable",
				method: http.MethodGet,
				path:   "/api/v1/me/drafts",
				configure: func(service *fakeContentService) {
					service.listAuthorDraftsErr = application.ErrDependencyUnavailable
				},
				wantStatus: http.StatusServiceUnavailable,
				wantCode:   1004,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				service := &fakeContentService{}
				if tc.configure != nil {
					tc.configure(service)
				}
				body := bytes.NewBufferString(tc.body)
				req := withUserID(httptest.NewRequest(tc.method, tc.path, body), "42")
				if tc.body != "" {
					req = withJSON(req)
				}
				rr := httptest.NewRecorder()

				NewHandler(service).ServeHTTP(rr, req)

				assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
			})
		}
	})
}
