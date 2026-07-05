package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/application"
)

func TestCommentDetailParsesPathAndViewer(t *testing.T) {
	now := time.Date(2026, 7, 5, 13, 0, 0, 0, time.UTC)
	service := &fakeCommentService{detailResult: application.CommentItem{
		PostID:          "post_pub_3",
		CommentID:       "c9001",
		RootCommentID:   "c9000",
		ParentCommentID: "c9000",
		Author:          application.AuthorSummary{PublicID: "u1", DisplayName: "Alice"},
		Content:         "reply",
		Status:          application.CommentStatusNormal,
		Stats:           application.CommentStats{LikeCount: 5, ReplyCount: 0},
		Viewer:          &application.ViewerState{Liked: true},
		CreatedAt:       now,
		UpdatedAt:       now,
	}}
	rr := httptest.NewRecorder()
	req := withUserID(httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_pub_3/comments/c9001", nil), "88")

	NewHandler(service).ServeHTTP(rr, req)

	if service.detailQuery.PostID != "post_pub_3" || service.detailQuery.CommentID != "c9001" || service.detailQuery.ViewerUserID != 88 {
		t.Fatalf("detail query = %#v", service.detailQuery)
	}
	var body envelope[commentItemResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Data.CommentID != "c9001" || body.Data.Viewer == nil || !body.Data.Viewer.Liked {
		t.Fatalf("status=%d body=%s decoded=%#v", rr.Code, rr.Body.String(), body)
	}
}

func TestRepliesPageParsesQueryAndReturnsPage(t *testing.T) {
	now := time.Date(2026, 7, 5, 13, 30, 0, 0, time.UTC)
	service := &fakeCommentService{repliesResult: application.CommentPage{
		Items: []application.CommentItem{{
			PostID:          "post_pub_4",
			CommentID:       "c9101",
			RootCommentID:   "c9100",
			ParentCommentID: "c9100",
			Status:          application.CommentStatusNormal,
			CreatedAt:       now,
			UpdatedAt:       now,
		}},
		Page: 2, Size: 10, Total: 21, Pages: 3,
	}}
	rr := httptest.NewRecorder()
	req := withUserID(httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_pub_4/comments/c9100/replies/page?page=2&size=10&sort=TIME", nil), "88")

	NewHandler(service).ServeHTTP(rr, req)

	if service.repliesQuery.PostID != "post_pub_4" || service.repliesQuery.RootCommentID != "c9100" || service.repliesQuery.Page != 2 || service.repliesQuery.Size != 10 || service.repliesQuery.Sort != application.CommentSortTime {
		t.Fatalf("replies query = %#v", service.repliesQuery)
	}
	var body envelope[commentPageResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Data.Total != 21 || len(body.Data.Items) != 1 {
		t.Fatalf("status=%d body=%s decoded=%#v", rr.Code, rr.Body.String(), body)
	}
}

func TestDeleteCommentRequiresLoginAndMapsResult(t *testing.T) {
	now := time.Date(2026, 7, 5, 14, 0, 0, 0, time.UTC)
	service := &fakeCommentService{deleteResult: application.DeleteCommentResult{
		PostID:        "post_pub_5",
		CommentID:     "c9201",
		DeletedAt:     now,
		DeletedByRole: application.DeletedByRoleAuthor,
		AffectedCount: 2,
	}}

	missingLogin := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(missingLogin, httptest.NewRequest(http.MethodDelete, "/api/v1/posts/post_pub_5/comments/c9201", nil))
	assertErrorEnvelope(t, missingLogin, http.StatusUnauthorized, 2006)

	rr := httptest.NewRecorder()
	req := withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/posts/post_pub_5/comments/c9201", nil), "42")
	NewHandler(service).ServeHTTP(rr, req)

	if service.deleteCmd.ActorUserID != 42 || service.deleteCmd.PostID != "post_pub_5" || service.deleteCmd.CommentID != "c9201" {
		t.Fatalf("delete command = %#v", service.deleteCmd)
	}
	var body envelope[deleteCommentResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Data.AffectedCount != 2 || body.Data.DeletedByRole != "AUTHOR" {
		t.Fatalf("status=%d body=%s decoded=%#v", rr.Code, rr.Body.String(), body)
	}
}

func TestAdminDeleteCommentRequiresAdminRoleAndBodyReason(t *testing.T) {
	service := &fakeCommentService{deleteResult: application.DeleteCommentResult{
		PostID:         "post_pub_5",
		CommentID:      "c9201",
		DeletedByRole:  application.DeletedByRoleAdmin,
		AffectedCount:  0,
		AlreadyDeleted: true,
	}}

	forbidden := httptest.NewRecorder()
	req := withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/admin/comments/posts/post_pub_5/comments/c9201", bytes.NewBufferString(`{"reason":"spam"}`)), "7")
	NewHandler(service).ServeHTTP(forbidden, req)
	assertErrorEnvelope(t, forbidden, http.StatusForbidden, 2007)

	rr := httptest.NewRecorder()
	req = withAdmin(withUserID(withJSON(httptest.NewRequest(http.MethodDelete, "/api/v1/admin/comments/posts/post_pub_5/comments/c9201", bytes.NewBufferString(`{"reason":"spam"}`))), "7"))
	NewHandler(service).ServeHTTP(rr, req)

	if service.adminDeleteCmd.ActorUserID != 7 || service.adminDeleteCmd.Reason != "spam" {
		t.Fatalf("admin delete command = %#v", service.adminDeleteCmd)
	}
	var body envelope[deleteCommentResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || !body.Data.AlreadyDeleted || body.Data.DeletedByRole != "ADMIN" {
		t.Fatalf("status=%d body=%s decoded=%#v", rr.Code, rr.Body.String(), body)
	}
}

func TestLikeUnlikeAndStatusRoutesUseTrustedActor(t *testing.T) {
	now := time.Date(2026, 7, 5, 15, 0, 0, 0, time.UTC)
	service := &fakeCommentService{
		likeResult:       application.LikeCommentResult{PostID: "post_pub_6", CommentID: "c9301", Liked: true, Changed: true, OccurredAt: now},
		unlikeResult:     application.LikeCommentResult{PostID: "post_pub_6", CommentID: "c9301", Liked: false, Changed: true, OccurredAt: now},
		likeStatusResult: application.LikeStatusResult{PostID: "post_pub_6", CommentID: "c9301", Liked: true},
	}

	NewHandler(service).ServeHTTP(httptest.NewRecorder(), withUserID(httptest.NewRequest(http.MethodPost, "/api/v1/posts/post_pub_6/comments/c9301/like", nil), "77"))
	NewHandler(service).ServeHTTP(httptest.NewRecorder(), withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/posts/post_pub_6/comments/c9301/like", nil), "77"))
	rr := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(rr, withUserID(httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_pub_6/comments/c9301/liked", nil), "77"))

	if service.likeCmd.ActorUserID != 77 || service.likeCmd.CommentID != "c9301" || service.unlikeCmd.ActorUserID != 77 || service.likeStatusQuery.ViewerUserID != 77 {
		t.Fatalf("commands: like=%#v unlike=%#v status=%#v", service.likeCmd, service.unlikeCmd, service.likeStatusQuery)
	}
	var body envelope[likeStatusResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || !body.Data.Liked {
		t.Fatalf("status=%d body=%s decoded=%#v", rr.Code, rr.Body.String(), body)
	}
}

func withAdmin(req *http.Request) *http.Request {
	req.Header.Set("X-User-Roles", "USER,ADMIN")
	return req
}
