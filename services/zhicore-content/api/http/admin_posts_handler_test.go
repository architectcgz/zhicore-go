package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
)

func TestAdminPostsRequiresAdminRole(t *testing.T) {
	service := &fakeContentService{}
	rr := httptest.NewRecorder()
	req := withUserID(httptest.NewRequest(http.MethodGet, "/api/v1/admin/content/posts?status=published", nil), "42")

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusForbidden, 2007)
	if service.listAdminPostsCalls != 0 {
		t.Fatalf("listAdminPostsCalls = %d, want 0", service.listAdminPostsCalls)
	}
}

func TestAdminPostsListMapsQueryAndResponse(t *testing.T) {
	publishedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	service := &fakeContentService{listAdminPostsResult: application.ListAdminPostsResult{
		Items: []application.AdminPostItem{{
			PostID:             "post_1",
			AuthorID:           "42",
			AuthorName:         "architect",
			AuthorAvatarFileID: "file_avatar",
			Title:              "Published title",
			Summary:            "summary",
			CoverFileID:        "file_cover",
			Status:             "PUBLISHED",
			PostVersion:        6,
			PublishedAt:        publishedAt,
			CreatedAt:          publishedAt.Add(-time.Hour),
			UpdatedAt:          publishedAt.Add(time.Minute),
			Stats:              application.PostStats{ViewCount: 10, LikeCount: 2, FavoriteCount: 1, CommentCount: 3},
		}},
		Page:  2,
		Size:  20,
		Total: 21,
	}}
	rr := httptest.NewRecorder()
	req := withRoles(withUserID(httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/content/posts?status=published&authorId=42&page=2&size=20",
		nil,
	), "1001"), "writer,admin")

	NewHandler(service).ServeHTTP(rr, req)

	if service.listAdminPostsCalls != 1 {
		t.Fatalf("listAdminPostsCalls = %d, want 1", service.listAdminPostsCalls)
	}
	if service.listAdminPostsQuery.Actor == nil || service.listAdminPostsQuery.Actor.UserID != 1001 ||
		!service.listAdminPostsQuery.Actor.HasRole("admin") {
		t.Fatalf("actor = %#v, want admin 1001", service.listAdminPostsQuery.Actor)
	}
	if service.listAdminPostsQuery.Status != "published" || service.listAdminPostsQuery.AuthorID != 42 ||
		service.listAdminPostsQuery.Page != 2 || service.listAdminPostsQuery.Size != 20 {
		t.Fatalf("query = %+v", service.listAdminPostsQuery)
	}

	var body envelope[adminPostListResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Data.Total != 21 || len(body.Data.Items) != 1 {
		t.Fatalf("status=%d body=%#v raw=%s", rr.Code, body, rr.Body.String())
	}
	item := body.Data.Items[0]
	if item.PostID != "post_1" || item.AuthorID != "42" || item.Title != "Published title" ||
		item.Stats.ViewCount != 10 || item.PublishedAt != formatTime(publishedAt) {
		t.Fatalf("item = %+v", item)
	}
}

func TestAdminPostDeleteMapsTrustedActorAndIgnoresBodyActor(t *testing.T) {
	service := &fakeContentService{deleteAdminPostResult: application.DeleteAdminPostResult{
		PostID: "post_1",
		Status: "DELETED",
	}}
	rr := httptest.NewRecorder()
	req := withRoles(withUserID(withJSON(httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/admin/content/posts/post_1",
		bytes.NewBufferString(`{"userId":999,"actor":{"userId":999},"reason":"policy violation"}`),
	)), "1001"), "ROLE_ADMIN")

	NewHandler(service).ServeHTTP(rr, req)

	if service.deleteAdminPostCalls != 1 {
		t.Fatalf("deleteAdminPostCalls = %d, want 1", service.deleteAdminPostCalls)
	}
	if service.deleteAdminPostCommand.Actor == nil || service.deleteAdminPostCommand.Actor.UserID != 1001 ||
		!service.deleteAdminPostCommand.Actor.HasRole("role_admin") {
		t.Fatalf("actor = %#v, want trusted admin 1001", service.deleteAdminPostCommand.Actor)
	}
	if service.deleteAdminPostCommand.PostID != "post_1" || service.deleteAdminPostCommand.Reason != "policy violation" {
		t.Fatalf("command = %+v", service.deleteAdminPostCommand)
	}

	var body envelope[adminPostDeleteResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Code != 200 || body.Data.PostID != "post_1" || body.Data.Status != "DELETED" {
		t.Fatalf("status=%d body=%#v raw=%s", rr.Code, body, rr.Body.String())
	}
}

func TestAdminPostDeleteAllowsEmptyBodyAndMapsErrors(t *testing.T) {
	service := &fakeContentService{
		deleteAdminPostResult: application.DeleteAdminPostResult{PostID: "post_1", Status: "DELETED"},
	}
	rr := httptest.NewRecorder()
	req := withRoles(withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/admin/content/posts/post_1", nil), "1001"), "admin")

	NewHandler(service).ServeHTTP(rr, req)

	assertSuccessCode(t, rr)
	if service.deleteAdminPostCommand.Reason != "" {
		t.Fatalf("reason = %q, want empty default handled by application", service.deleteAdminPostCommand.Reason)
	}

	service = &fakeContentService{
		deleteAdminPostResult: application.DeleteAdminPostResult{PostID: "post_1", Status: "DELETED"},
	}
	rr = httptest.NewRecorder()
	req = withRoles(withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/admin/content/posts/post_1", http.NoBody), "1001"), "admin")
	req.ContentLength = -1

	NewHandler(service).ServeHTTP(rr, req)

	assertSuccessCode(t, rr)
	if service.deleteAdminPostCalls != 1 {
		t.Fatalf("deleteAdminPostCalls = %d, want 1 for unknown-length empty body", service.deleteAdminPostCalls)
	}

	service = &fakeContentService{deleteAdminPostErr: application.ErrPostDeleted}
	rr = httptest.NewRecorder()
	req = withRoles(withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/admin/content/posts/post_1", nil), "1001"), "admin")

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusConflict, 4004)
}
