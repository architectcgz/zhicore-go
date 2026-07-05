package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
)

func TestListPublishedPostsHandler(t *testing.T) {
	publishedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	service := &fakeContentService{
		listPublishedResult: application.ListPublishedPostsResult{
			Items: []application.PostSummary{{
				PostID:      "post_1",
				AuthorID:    "42",
				Title:       "Published",
				Status:      "PUBLISHED",
				PublishedAt: publishedAt,
				CreatedAt:   publishedAt.Add(-time.Hour),
				UpdatedAt:   publishedAt,
				Stats:       application.PostStats{ViewCount: 10, LikeCount: 2},
			}},
			NextCursor: "cursor_next",
			HasMore:    true,
			Limit:      10,
		},
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?authorId=42&cursor=cursor_1&limit=10&sort=latest", nil)

	NewHandler(service).ServeHTTP(rr, req)

	if service.listPublishedCalls != 1 {
		t.Fatalf("list calls = %d, want 1", service.listPublishedCalls)
	}
	if service.listPublishedQuery.AuthorID != "42" || service.listPublishedQuery.Cursor != "cursor_1" || service.listPublishedQuery.Limit != 10 || service.listPublishedQuery.Sort != "latest" {
		t.Fatalf("query = %+v", service.listPublishedQuery)
	}
	var body envelope[cursorPageResp[postSummaryResp]]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || len(body.Data.Items) != 1 || body.Data.Items[0].PostID != "post_1" {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !body.Data.HasMore || body.Data.NextCursor != "cursor_next" || body.Data.Limit != 10 {
		t.Fatalf("page = %+v", body.Data)
	}
}

func TestListPublishedPostsHandlerMapsInvalidQuery(t *testing.T) {
	service := &fakeContentService{listPublishedErr: application.ErrInvalidArgument}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts?cursor=bad", nil)

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusBadRequest, 1001)
}

func TestGetPostDetailHandler(t *testing.T) {
	publishedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	service := &fakeContentService{
		getDetailResult: application.GetPostDetailResult{
			Post: application.PostSummary{
				PostID:      "post_1",
				AuthorID:    "42",
				Title:       "Published",
				Status:      "PUBLISHED",
				PublishedAt: publishedAt,
				CreatedAt:   publishedAt.Add(-time.Hour),
				UpdatedAt:   publishedAt,
				Stats:       application.PostStats{ViewCount: 10},
			},
			Body: &application.PostBodyResult{
				BodyID:        "body_1",
				SchemaVersion: 1,
				CanonicalJSON: []byte(`{"schemaVersion":1,"blocks":[]}`),
				PlainText:     "published body",
				ContentHash:   "sha256:body",
				SizeBytes:     36,
				CreatedAt:     publishedAt,
			},
		},
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_1", nil)

	NewHandler(service).ServeHTTP(rr, req)

	if service.getDetailCalls != 1 || service.getDetailQuery.PostID != "post_1" {
		t.Fatalf("detail calls/query = %d/%+v", service.getDetailCalls, service.getDetailQuery)
	}
	var body envelope[postDetailResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Data.Post.PostID != "post_1" || body.Data.Body == nil || body.Data.Body.BodyID != "body_1" {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetPostDetailHandlerMapsNotFound(t *testing.T) {
	service := &fakeContentService{getDetailErr: application.ErrPostNotFound}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_missing", nil)

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusNotFound, 4001)
}

func TestBatchGetPublishedPostsHandler(t *testing.T) {
	service := &fakeContentService{
		batchResult: application.BatchGetPublishedPostsResult{
			Items:          []application.PostSummary{{PostID: "post_1", AuthorID: "42", Title: "Published", Status: "PUBLISHED"}},
			MissingPostIDs: []string{"post_missing"},
		},
	}
	rr := httptest.NewRecorder()
	req := withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/batch-get", bytes.NewBufferString(`{
		"postIds":["post_1","post_missing"],
		"includeDeleted":true
	}`)))

	NewHandler(service).ServeHTTP(rr, req)

	if service.batchCalls != 1 || len(service.batchQuery.PostIDs) != 2 || service.batchQuery.IncludeDeleted {
		t.Fatalf("batch query = %+v, includeDeleted must not be enabled for anonymous calls", service.batchQuery)
	}
	var body envelope[batchGetPostsResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || len(body.Data.Items) != 1 || len(body.Data.MissingPostIDs) != 1 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestBatchGetPublishedPostsHandlerRejectsInvalidBody(t *testing.T) {
	service := &fakeContentService{}
	rr := httptest.NewRecorder()
	req := withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/batch-get", bytes.NewBufferString(`{"postIds":[]}`)))

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusBadRequest, 1001)
	if service.batchCalls != 0 {
		t.Fatalf("batch calls = %d, want none", service.batchCalls)
	}
}
