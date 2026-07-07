package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
)

func TestEngagementCommandHandlersRequireTrustedActorAndReturnState(t *testing.T) {
	t.Run("like requires actor", func(t *testing.T) {
		service := &fakeContentService{}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_1/like", nil)

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
		if service.likePostCalls != 0 {
			t.Fatalf("like calls = %d, want 0", service.likePostCalls)
		}
	})

	t.Run("like uses trusted actor", func(t *testing.T) {
		service := &fakeContentService{likePostResult: application.EngagementResult{
			PostID:    "post_1",
			Liked:     true,
			Favorited: false,
			Stats:     application.PostStats{LikeCount: 8, FavoriteCount: 2},
		}}
		rr := httptest.NewRecorder()
		req := withUserID(httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_1/like", nil), "42")

		NewHandler(service).ServeHTTP(rr, req)

		if service.likePostCalls != 1 || service.likePostCmd.Actor.UserID != 42 || service.likePostCmd.PostID != "post_1" {
			t.Fatalf("like command = %+v calls=%d", service.likePostCmd, service.likePostCalls)
		}
		var body envelope[engagementMutationResp]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || body.Data.PostID != "post_1" || !body.Data.Liked || body.Data.Favorited || body.Data.Stats.LikeCount != 8 {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}

func TestGetPostEngagementHandler(t *testing.T) {
	t.Run("anonymous response omits viewer", func(t *testing.T) {
		service := &fakeContentService{getEngagementResult: application.PostEngagementResult{
			PostID: "post_1",
			Stats:  application.PostStats{ViewCount: 10, LikeCount: 3},
		}}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_1/engagement", nil)

		NewHandler(service).ServeHTTP(rr, req)

		if service.getEngagementCalls != 1 || service.getEngagementQuery.Actor != nil {
			t.Fatalf("query = %+v calls=%d", service.getEngagementQuery, service.getEngagementCalls)
		}
		if bytes.Contains(rr.Body.Bytes(), []byte(`"viewer"`)) {
			t.Fatalf("anonymous body must omit viewer: %s", rr.Body.String())
		}
		var body envelope[postEngagementResp]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || body.Data.Stats.ViewCount != 10 || body.Data.Viewer != nil {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("logged in response maps known viewer", func(t *testing.T) {
		service := &fakeContentService{getEngagementResult: application.PostEngagementResult{
			PostID: "post_1",
			Stats:  application.PostStats{LikeCount: 3, FavoriteCount: 2},
			Viewer: &application.EngagementViewer{
				Liked:     application.KnownBool(true),
				Favorited: application.KnownBool(false),
			},
		}}
		rr := httptest.NewRecorder()
		req := withUserID(httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_1/engagement", nil), "42")

		NewHandler(service).ServeHTTP(rr, req)

		if service.getEngagementQuery.Actor == nil || service.getEngagementQuery.Actor.UserID != 42 {
			t.Fatalf("query actor = %+v", service.getEngagementQuery.Actor)
		}
		var body envelope[postEngagementResp]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || body.Data.Viewer == nil || body.Data.Viewer.Liked == nil || !*body.Data.Viewer.Liked ||
			body.Data.Viewer.Favorited == nil || *body.Data.Viewer.Favorited || body.Data.Viewer.Degraded {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("degraded viewer preserves null states", func(t *testing.T) {
		service := &fakeContentService{getEngagementResult: application.PostEngagementResult{
			PostID: "post_1",
			Stats:  application.PostStats{LikeCount: 3},
			Viewer: &application.EngagementViewer{
				Liked:     application.UnknownBool(),
				Favorited: application.UnknownBool(),
				Degraded:  true,
			},
		}}
		rr := httptest.NewRecorder()
		req := withUserID(httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_1/engagement", nil), "42")

		NewHandler(service).ServeHTTP(rr, req)

		if !bytes.Contains(rr.Body.Bytes(), []byte(`"liked":null`)) || !bytes.Contains(rr.Body.Bytes(), []byte(`"favorited":null`)) {
			t.Fatalf("degraded viewer must preserve null states: %s", rr.Body.String())
		}
		var body envelope[postEngagementResp]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || body.Data.Viewer == nil || body.Data.Viewer.Liked != nil || body.Data.Viewer.Favorited != nil || !body.Data.Viewer.Degraded {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}

func TestBatchGetEngagementStatusHandler(t *testing.T) {
	service := &fakeContentService{batchEngagementResult: application.BatchEngagementStatusResult{
		Items: []application.EngagementStatusItem{
			{PostID: "post_1", Liked: application.KnownBool(true), Favorited: application.KnownBool(false)},
			{PostID: "post_2", Liked: application.UnknownBool(), Favorited: application.UnknownBool(), Degraded: true},
		},
	}}
	rr := httptest.NewRecorder()
	req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/engagement/batch-status", bytes.NewBufferString(`{
		"postIds":["post_1","post_2"]
	}`))), "42")

	NewHandler(service).ServeHTTP(rr, req)

	if service.batchEngagementCalls != 1 || service.batchEngagementQuery.Actor.UserID != 42 || len(service.batchEngagementQuery.PostIDs) != 2 {
		t.Fatalf("query = %+v calls=%d", service.batchEngagementQuery, service.batchEngagementCalls)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"liked":null`)) || !bytes.Contains(rr.Body.Bytes(), []byte(`"degraded":true`)) {
		t.Fatalf("batch degraded item must preserve null and degraded: %s", rr.Body.String())
	}
	var body envelope[batchEngagementStatusResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || len(body.Data.Items) != 2 || body.Data.Items[0].Liked == nil || !*body.Data.Items[0].Liked ||
		body.Data.Items[1].Liked != nil || !body.Data.Items[1].Degraded {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestBatchGetEngagementStatusHandlerRejectsInvalidRequests(t *testing.T) {
	t.Run("requires login", func(t *testing.T) {
		service := &fakeContentService{}
		rr := httptest.NewRecorder()
		req := withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/engagement/batch-status", bytes.NewBufferString(`{"postIds":["post_1"]}`)))

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
	})

	t.Run("rejects empty post ids", func(t *testing.T) {
		service := &fakeContentService{}
		rr := httptest.NewRecorder()
		req := withUserID(withJSON(httptest.NewRequest(http.MethodPost, "/api/v1/posts/engagement/batch-status", bytes.NewBufferString(`{"postIds":[]}`))), "42")

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusBadRequest, 1001)
		if service.batchEngagementCalls != 0 {
			t.Fatalf("batch calls = %d, want 0", service.batchEngagementCalls)
		}
	})
}
