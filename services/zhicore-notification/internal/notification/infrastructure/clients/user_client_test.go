package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func TestUserFollowerClientListsFollowerShardWithNotificationCaller(t *testing.T) {
	var gotRequest usercontract.ListFollowerShardRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != usercontract.ListFollowerShardPath {
			t.Fatalf("path = %s, want %s", r.URL.Path, usercontract.ListFollowerShardPath)
		}
		if r.Header.Get("X-Caller-Service") != "zhicore-notification" ||
			r.Header.Get("X-Caller-Operation") != usercontract.OperationNotificationListFollowerShard {
			t.Fatalf("caller headers = %q/%q", r.Header.Get("X-Caller-Service"), r.Header.Get("X-Caller-Operation"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 200,
			"data": usercontract.ListFollowerShardResponse{
				FollowerIDs: []int64{101, 102},
				NextCursor:  "cursor-2",
				HasMore:     true,
			},
		})
	}))
	t.Cleanup(server.Close)
	activeSince := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	client := NewUserFollowerClient(UserFollowerClientConfig{BaseURL: server.URL, HTTPClient: server.Client()})

	page, err := client.ListFollowerShard(context.Background(), ports.ListFollowerShardInput{
		FollowingID:   77,
		AudienceClass: "HOT",
		ActiveSince:   &activeSince,
		Cursor:        "cursor-1",
		Limit:         200,
	})
	if err != nil {
		t.Fatalf("ListFollowerShard() error = %v", err)
	}
	if gotRequest.FollowingID != 77 ||
		gotRequest.AudienceClass != "HOT" ||
		gotRequest.ActiveSince != "2026-07-01T00:00:00Z" ||
		gotRequest.Cursor != "cursor-1" ||
		gotRequest.Limit != 200 {
		t.Fatalf("request = %#v", gotRequest)
	}
	if len(page.FollowerIDs) != 2 || page.FollowerIDs[0] != 101 || page.NextCursor != "cursor-2" || !page.HasMore {
		t.Fatalf("page = %#v", page)
	}
}

func TestUserFollowerClientMapsProviderFailureToDependencyUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 1004})
	}))
	t.Cleanup(server.Close)
	client := NewUserFollowerClient(UserFollowerClientConfig{BaseURL: server.URL, HTTPClient: server.Client()})

	_, err := client.ListFollowerShard(context.Background(), ports.ListFollowerShardInput{FollowingID: 77, AudienceClass: "HOT", Limit: 200})
	if err != ports.ErrDependencyUnavailable {
		t.Fatalf("ListFollowerShard() error = %v, want dependency unavailable", err)
	}
}
