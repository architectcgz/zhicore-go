package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
)

func TestReaderSessionHandlers(t *testing.T) {
	t.Run("anonymous heartbeat is allowed no-op", func(t *testing.T) {
		service := &fakeContentService{upsertPresenceResult: application.ReaderPresenceResult{
			PostID: "post_1", OnlineCount: 0, Degraded: true, TTLSeconds: 30,
		}}
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_1/reader-sessions/sess_1", nil)

		NewHandler(service).ServeHTTP(rr, req)

		if service.upsertPresenceCalls != 1 || service.upsertPresenceCmd.Actor != nil || service.upsertPresenceCmd.SessionID != "sess_1" {
			t.Fatalf("command = %+v calls=%d", service.upsertPresenceCmd, service.upsertPresenceCalls)
		}
		var body envelope[readerPresenceResp]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || !body.Data.Degraded || body.Data.OnlineCount != 0 || body.Data.TTLSeconds != 30 {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("logged in heartbeat uses trusted actor", func(t *testing.T) {
		service := &fakeContentService{upsertPresenceResult: application.ReaderPresenceResult{
			PostID: "post_1", OnlineCount: 2, TTLSeconds: 30,
		}}
		rr := httptest.NewRecorder()
		req := withUserID(httptest.NewRequest(http.MethodPut, "/api/v1/posts/post_1/reader-sessions/sess_1", nil), "42")

		NewHandler(service).ServeHTTP(rr, req)

		if service.upsertPresenceCmd.Actor == nil || service.upsertPresenceCmd.Actor.UserID != 42 {
			t.Fatalf("actor = %+v", service.upsertPresenceCmd.Actor)
		}
		var body envelope[readerPresenceResp]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if rr.Code != http.StatusOK || body.Data.OnlineCount != 2 || body.Data.Degraded {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("leave returns empty success", func(t *testing.T) {
		service := &fakeContentService{}
		rr := httptest.NewRecorder()
		req := withUserID(httptest.NewRequest(http.MethodDelete, "/api/v1/posts/post_1/reader-sessions/sess_1", nil), "42")

		NewHandler(service).ServeHTTP(rr, req)

		if service.deletePresenceCalls != 1 || service.deletePresenceCmd.Actor.UserID != 42 || service.deletePresenceCmd.SessionID != "sess_1" {
			t.Fatalf("command = %+v calls=%d", service.deletePresenceCmd, service.deletePresenceCalls)
		}
		if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte(`"data":{}`)) {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})
}

func TestGetReaderPresenceHandler(t *testing.T) {
	service := &fakeContentService{getPresenceResult: application.ReaderPresenceResult{
		PostID: "post_1", OnlineCount: 0, Degraded: true, TTLSeconds: 30,
	}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/post_1/reader-presence", nil)

	NewHandler(service).ServeHTTP(rr, req)

	if service.getPresenceCalls != 1 || service.getPresenceQuery.PostID != "post_1" {
		t.Fatalf("query = %+v calls=%d", service.getPresenceQuery, service.getPresenceCalls)
	}
	var body envelope[readerPresenceResp]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if rr.Code != http.StatusOK || body.Data.PostID != "post_1" || !body.Data.Degraded || body.Data.TTLSeconds != 30 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
