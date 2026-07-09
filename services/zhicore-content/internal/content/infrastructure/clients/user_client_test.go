package clients

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestUserClientGetOwnerSnapshot(t *testing.T) {
	t.Run("gets user snapshot through provider contract", func(t *testing.T) {
		var gotPath string
		var gotCallerService string
		var gotCallerOperation string
		var gotRequest usercontract.IDsRequest
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotCallerService = r.Header.Get("X-Caller-Service")
			gotCallerOperation = r.Header.Get("X-Caller-Operation")
			if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
				t.Fatalf("Decode request: %v", err)
			}
			writeSuccess(t, w, usercontract.SimpleBatchResponse{
				Items: []usercontract.SimpleUser{{
					UserID:         42,
					Nickname:       "architect",
					AvatarFileID:   "avatar_1",
					ProfileVersion: 7,
					Status:         "ACTIVE",
				}},
			})
		}))
		defer server.Close()
		client := NewUserClient(UserClientConfig{BaseURL: server.URL, HTTPClient: server.Client()})

		got, err := client.GetOwnerSnapshot(context.Background(), 42)
		if err != nil {
			t.Fatalf("GetOwnerSnapshot() error = %v", err)
		}

		if gotPath != usercontract.BatchSimplePath {
			t.Fatalf("path = %q, want %q", gotPath, usercontract.BatchSimplePath)
		}
		if gotCallerService != "zhicore-content" || gotCallerOperation != usercontract.OperationContentGetOwnerSnapshot {
			t.Fatalf("caller headers = %q/%q", gotCallerService, gotCallerOperation)
		}
		if len(gotRequest.UserIDs) != 1 || gotRequest.UserIDs[0] != 42 {
			t.Fatalf("request = %+v, want user 42", gotRequest)
		}
		if got.DisplayName != "architect" || got.AvatarFileID != "avatar_1" || got.ProfileVersion != 7 {
			t.Fatalf("snapshot = %+v, want provider DTO mapped", got)
		}
	})

	t.Run("maps missing user and provider failures without leaking base URL", func(t *testing.T) {
		tests := []struct {
			name       string
			statusCode int
			body       any
		}{
			{name: "missing user", statusCode: http.StatusOK, body: usercontract.SimpleBatchResponse{MissingUserIDs: []int64{42}}},
			{name: "http not found", statusCode: http.StatusNotFound, body: nil},
			{name: "server error", statusCode: http.StatusInternalServerError, body: nil},
			{name: "envelope error", statusCode: http.StatusOK, body: errorEnvelope{Code: 1004, Message: "degraded"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tt.body == nil {
						w.WriteHeader(tt.statusCode)
						return
					}
					if envelope, ok := tt.body.(errorEnvelope); ok {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(tt.statusCode)
						_ = json.NewEncoder(w).Encode(envelope)
						return
					}
					writeSuccess(t, w, tt.body)
				}))
				defer server.Close()
				rawBaseURL := server.URL + "/?token=secret"
				client := NewUserClient(UserClientConfig{BaseURL: rawBaseURL, HTTPClient: server.Client()})

				_, err := client.GetOwnerSnapshot(context.Background(), 42)

				if !errors.Is(err, ports.ErrDependencyUnavailable) {
					t.Fatalf("error = %v, want ErrDependencyUnavailable", err)
				}
				assertNoSecretLeak(t, err)
			})
		}
	})

	t.Run("maps timeout and context cancellation to dependency errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
		}))
		defer server.Close()
		client := NewUserClient(UserClientConfig{
			BaseURL:    server.URL,
			HTTPClient: server.Client(),
			Timeout:    time.Millisecond,
		})

		_, err := client.GetOwnerSnapshot(context.Background(), 42)
		if !errors.Is(err, ports.ErrDependencyUnavailable) {
			t.Fatalf("timeout error = %v, want ErrDependencyUnavailable", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = client.GetOwnerSnapshot(ctx, 42)
		if !errors.Is(err, ports.ErrDependencyUnavailable) || !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel error = %v, want dependency and context canceled", err)
		}
	})

	t.Run("retries transient provider failure up to configured attempts", func(t *testing.T) {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			writeSuccess(t, w, usercontract.SimpleBatchResponse{
				Items: []usercontract.SimpleUser{{
					UserID:         42,
					Nickname:       "architect",
					ProfileVersion: 8,
				}},
			})
		}))
		defer server.Close()
		client := NewUserClient(UserClientConfig{BaseURL: server.URL, HTTPClient: server.Client(), MaxAttempts: 2})

		got, err := client.GetOwnerSnapshot(context.Background(), 42)
		if err != nil {
			t.Fatalf("GetOwnerSnapshot() error = %v", err)
		}
		if calls != 2 {
			t.Fatalf("provider calls = %d, want 2", calls)
		}
		if got.DisplayName != "architect" || got.ProfileVersion != 8 {
			t.Fatalf("snapshot = %+v, want retried provider DTO", got)
		}
	})
}

func writeSuccess(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	}{Code: 200, Message: "操作成功", Data: data})
}

type errorEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func assertNoSecretLeak(t *testing.T, err error) {
	t.Helper()
	text := err.Error()
	if strings.Contains(text, "secret") || strings.Contains(text, "token=") {
		t.Fatalf("error leaked sensitive URL: %v", err)
	}
}
