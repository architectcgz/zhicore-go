package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
)

func TestTagQueryHandlers(t *testing.T) {
	t.Run("list tags", func(t *testing.T) {
		service := &fakeContentService{listTagsResult: application.TagPageResult{
			Items:   []application.Tag{{TagID: "tag_go", Name: "Go", Slug: "go", PostCount: 8}},
			HasMore: true,
			Limit:   10,
		}}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags?limit=10", nil)

		NewHandler(service).ServeHTTP(rr, req)

		if service.listTagsCalls != 1 || service.listTagsQuery.Limit != 10 {
			t.Fatalf("list query = %+v calls=%d", service.listTagsQuery, service.listTagsCalls)
		}
		var body envelope[cursorPageResp[tagResp]]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || body.Code != 200 || len(body.Data.Items) != 1 || body.Data.Items[0].Slug != "go" {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("get tag maps not found", func(t *testing.T) {
		service := &fakeContentService{getTagErr: application.ErrTaxonomyReferenceNotFound}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/missing", nil)

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusNotFound, 4012)
	})

	t.Run("search tags", func(t *testing.T) {
		service := &fakeContentService{searchTagsResult: []application.Tag{{TagID: "tag_go", Name: "Go", Slug: "go", PostCount: 8}}}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/search?q=go&limit=5", nil)

		NewHandler(service).ServeHTTP(rr, req)

		if service.searchTagsCalls != 1 || service.searchTagsQuery.Query != "go" || service.searchTagsQuery.Limit != 5 {
			t.Fatalf("search query = %+v calls=%d", service.searchTagsQuery, service.searchTagsCalls)
		}
		var body envelope[[]tagResp]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || body.Code != 200 || len(body.Data) != 1 {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("hot tags", func(t *testing.T) {
		service := &fakeContentService{listHotTagsResult: []application.Tag{{TagID: "tag_go", Name: "Go", Slug: "go", PostCount: 8}}}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/hot", nil)

		NewHandler(service).ServeHTTP(rr, req)

		if service.listHotTagsCalls != 1 {
			t.Fatalf("hot calls = %d", service.listHotTagsCalls)
		}
		var body envelope[[]tagResp]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || len(body.Data) != 1 || body.Data[0].PostCount != 8 {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}

func TestListPostsByTagHandler(t *testing.T) {
	publishedAt := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	service := &fakeContentService{listPostsByTagResult: application.ListPublishedPostsResult{
		Items: []application.PostSummary{{PostID: "post_1", AuthorID: "42", Title: "Published", Status: "PUBLISHED", PublishedAt: publishedAt, CreatedAt: publishedAt, UpdatedAt: publishedAt}},
		Limit: 20,
	}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags/go/posts?limit=20", nil)

	NewHandler(service).ServeHTTP(rr, req)

	if service.listPostsByTagCalls != 1 || service.listPostsByTagQuery.Slug != "go" || service.listPostsByTagQuery.Limit != 20 {
		t.Fatalf("query = %+v calls=%d", service.listPostsByTagQuery, service.listPostsByTagCalls)
	}
	var body envelope[cursorPageResp[postSummaryResp]]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || len(body.Data.Items) != 1 || body.Data.Items[0].PostID != "post_1" {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPostTagsHandlers(t *testing.T) {
	t.Run("get post tags", func(t *testing.T) {
		service := &fakeContentService{getPostTagsResult: []application.Tag{{TagID: "tag_go", Name: "Go", Slug: "go", PostCount: 8}}}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_1/tags", nil)

		NewHandler(service).ServeHTTP(rr, req)

		if service.getPostTagsCalls != 1 || service.getPostTagsQuery.PostID != "post_1" {
			t.Fatalf("query = %+v calls=%d", service.getPostTagsQuery, service.getPostTagsCalls)
		}
		var body envelope[[]tagResp]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || len(body.Data) != 1 || body.Data[0].Slug != "go" {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("update post tags uses trusted actor", func(t *testing.T) {
		service := &fakeContentService{updatePostTagsResult: application.PostTagsMutationResult{
			PostID:      "post_1",
			PostVersion: 4,
			Tags:        []application.Tag{{TagID: "tag_go", Name: "Go", Slug: "go", PostCount: 8}},
			UpdatedAt:   time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
		}}
		rr := httptest.NewRecorder()
		req := withUserID(withJSON(httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_1/tags", bytes.NewBufferString(`{
			"userId":999,
			"basePostVersion":3,
			"tags":["Go","go"]
		}`))), "42")

		NewHandler(service).ServeHTTP(rr, req)

		if service.updatePostTagsCalls != 1 || service.updatePostTagsCmd.Actor.UserID != 42 || len(service.updatePostTagsCmd.Tags) != 2 {
			t.Fatalf("command = %+v calls=%d", service.updatePostTagsCmd, service.updatePostTagsCalls)
		}
		var body envelope[postTagsMutationResp]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || body.Data.PostVersion != 4 || len(body.Data.Tags) != 1 {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("update post tags requires tags field", func(t *testing.T) {
		service := &fakeContentService{}
		rr := httptest.NewRecorder()
		req := withUserID(withJSON(httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_1/tags", bytes.NewBufferString(`{
			"basePostVersion":3
		}`))), "42")

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusBadRequest, 1001)
		if service.updatePostTagsCalls != 0 {
			t.Fatalf("update calls = %d, want 0", service.updatePostTagsCalls)
		}
	})

	t.Run("delete post tag requires version", func(t *testing.T) {
		service := &fakeContentService{}
		rr := httptest.NewRecorder()
		req := withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/posts/post_1/tags/go", nil), "42")

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusBadRequest, 1001)
		if service.deletePostTagCalls != 0 {
			t.Fatalf("delete calls = %d, want 0", service.deletePostTagCalls)
		}
	})
}
