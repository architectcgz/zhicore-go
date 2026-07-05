package clients

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func TestContentClientCheckPostCommentableSendsCallerHeadersAndMapsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/internal/posts/post_pub_1/comment-context" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Caller-Service") != "zhicore-comment" || r.Header.Get("X-Caller-Operation") != "comment.check_post_commentable" {
			t.Fatalf("caller headers = %q/%q", r.Header.Get("X-Caller-Service"), r.Header.Get("X-Caller-Operation"))
		}
		writeEnvelope(t, w, http.StatusOK, 200, map[string]any{
			"postId":      "post_pub_1",
			"internalId":  float64(9001),
			"authorId":    float64(42),
			"commentable": true,
			"status":      "PUBLISHED",
		})
	}))
	defer server.Close()

	client := NewContentClient(Config{BaseURL: server.URL, HTTPClient: server.Client()})
	post, err := client.CheckPostCommentable(context.Background(), "post_pub_1")
	if err != nil {
		t.Fatalf("CheckPostCommentable() error = %v", err)
	}
	if post.PostID != "post_pub_1" || post.ContentInternalID != 9001 || post.AuthorID != 42 {
		t.Fatalf("post = %#v", post)
	}
}

func TestContentClientMapsPostNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusNotFound, 4001, nil)
	}))
	defer server.Close()

	client := NewContentClient(Config{BaseURL: server.URL, HTTPClient: server.Client()})
	_, err := client.CheckPostCommentable(context.Background(), "missing")
	if !errors.Is(err, ports.ErrPostNotFound) {
		t.Fatalf("CheckPostCommentable() error = %v, want ErrPostNotFound", err)
	}
}

func TestUserClientChecksAvailabilityAndBlockedPairs(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Header.Get("X-Caller-Service") != "zhicore-comment" {
			t.Fatalf("caller service = %q", r.Header.Get("X-Caller-Service"))
		}
		switch r.URL.Path {
		case usercontract.BatchAvailabilityPath:
			if r.Header.Get("X-Caller-Operation") != usercontract.OperationCommentCheckUserAvailability {
				t.Fatalf("operation = %q", r.Header.Get("X-Caller-Operation"))
			}
			writeEnvelope(t, w, http.StatusOK, 200, map[string]any{
				"items": []map[string]any{{"userId": float64(77), "available": true, "status": "ACTIVE"}},
			})
		case usercontract.BatchCheckBlockedPath:
			if r.Header.Get("X-Caller-Operation") != usercontract.OperationCommentBatchCheckBlocked {
				t.Fatalf("operation = %q", r.Header.Get("X-Caller-Operation"))
			}
			writeEnvelope(t, w, http.StatusOK, 200, map[string]any{
				"items": []map[string]any{{"blockerId": float64(42), "blockedId": float64(77), "blocked": true}},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewUserClient(Config{BaseURL: server.URL, HTTPClient: server.Client()})
	if err := client.EnsureUserCanComment(context.Background(), 77); err != nil {
		t.Fatalf("EnsureUserCanComment() error = %v", err)
	}
	blocked, err := client.BatchCheckBlocked(context.Background(), []ports.BlockPair{{BlockerID: 42, BlockedID: 77}})
	if err != nil {
		t.Fatalf("BatchCheckBlocked() error = %v", err)
	}
	if !blocked[ports.BlockPair{BlockerID: 42, BlockedID: 77}] || len(paths) != 2 {
		t.Fatalf("blocked=%#v paths=%#v", blocked, paths)
	}
}

func TestUserClientBatchGetAuthorSummariesReturnsMissingAsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != usercontract.BatchSimplePath || r.Header.Get("X-Caller-Operation") != usercontract.OperationCommentBatchGetAuthorSummaries {
			t.Fatalf("request path/op = %s/%s", r.URL.Path, r.Header.Get("X-Caller-Operation"))
		}
		writeEnvelope(t, w, http.StatusOK, 200, map[string]any{
			"items":          []map[string]any{{"userId": float64(77), "publicId": "user_pub_77", "nickname": "Alice", "avatarFileId": "avatar_1"}},
			"missingUserIds": []float64{88},
		})
	}))
	defer server.Close()

	client := NewUserClient(Config{BaseURL: server.URL, HTTPClient: server.Client()})
	summaries, err := client.BatchGetAuthorSummaries(context.Background(), []domain.UserID{77, 88})
	if err != nil {
		t.Fatalf("BatchGetAuthorSummaries() error = %v", err)
	}
	if summaries[77].PublicID != "user_pub_77" || summaries[77].DisplayName != "Alice" || !summaries[88].Unavailable {
		t.Fatalf("summaries = %#v", summaries)
	}
}

func TestFileClientEnsureCommentMediaReferencedPostsMediaGuard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/internal/files/comment-media/validate" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Caller-Operation") != "comment.ensure_comment_media_referenced" {
			t.Fatalf("operation = %q", r.Header.Get("X-Caller-Operation"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["voiceFileId"] != "voice_1" || int(body["voiceDuration"].(float64)) != 3 {
			t.Fatalf("body = %#v", body)
		}
		writeEnvelope(t, w, http.StatusOK, 200, nil)
	}))
	defer server.Close()

	client := NewFileClient(Config{BaseURL: server.URL, HTTPClient: server.Client()})
	err := client.EnsureCommentMediaReferenced(context.Background(), ports.CommentMediaReferences{VoiceFileID: "voice_1", VoiceDuration: 3})
	if err != nil {
		t.Fatalf("EnsureCommentMediaReferenced() error = %v", err)
	}
}

func writeEnvelope(t *testing.T, w http.ResponseWriter, status int, code int, data any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{"code": code, "message": "ok", "data": data}); err != nil {
		t.Fatalf("write response: %v", err)
	}
}
