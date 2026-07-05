package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientDoJSONWritesCallerHeadersAndDecodesEnvelope(t *testing.T) {
	var gotPath string
	var gotQuery string
	var gotCallerService string
	var gotCallerOperation string
	var gotRequest map[string]int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotCallerService = r.Header.Get("X-Caller-Service")
		gotCallerOperation = r.Header.Get("X-Caller-Operation")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("Decode request: %v", err)
		}
		writeEnvelope(t, w, http.StatusOK, 200, map[string]string{"name": "architect"})
	}))
	defer server.Close()
	client := New(Config{BaseURL: server.URL + "/api?token=secret", CallerService: "zhicore-content", HTTPClient: server.Client()})

	var got struct {
		Name string `json:"name"`
	}
	err := client.DoJSON(context.Background(), http.MethodPost, "/v1/users", "content.get_owner_snapshot", map[string]int{"id": 42}, &got)
	if err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}

	if gotPath != "/api/v1/users" || gotQuery != "" {
		t.Fatalf("request path/query = %q/%q, want sanitized joined path without query", gotPath, gotQuery)
	}
	if gotCallerService != "zhicore-content" || gotCallerOperation != "content.get_owner_snapshot" {
		t.Fatalf("caller headers = %q/%q", gotCallerService, gotCallerOperation)
	}
	if gotRequest["id"] != 42 || got.Name != "architect" {
		t.Fatalf("request=%+v response=%+v, want mapped request/response", gotRequest, got)
	}
}

func TestClientDoJSONReturnsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusNotFound, 4001, nil)
	}))
	defer server.Close()
	client := New(Config{BaseURL: server.URL, CallerService: "zhicore-content", HTTPClient: server.Client()})

	err := client.DoJSON(context.Background(), http.MethodPost, "/missing", "content.test", nil, nil)

	var providerErr ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %v, want ProviderError", err)
	}
	if providerErr.StatusCode != http.StatusNotFound || providerErr.Code != 4001 {
		t.Fatalf("provider error = %+v, want status/code", providerErr)
	}
}

func TestClientDoJSONReturnsProviderErrorForEmptyNon2xxBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	defer server.Close()
	client := New(Config{BaseURL: server.URL, CallerService: "zhicore-content", HTTPClient: server.Client()})

	err := client.DoJSON(context.Background(), http.MethodPost, "/gone", "content.test", nil, nil)

	var providerErr ProviderError
	if !errors.As(err, &providerErr) || providerErr.StatusCode != http.StatusGone {
		t.Fatalf("error = %v, want ProviderError with 410", err)
	}
}

func TestClientDoJSONTransportErrorsDoNotLeakRawBaseURL(t *testing.T) {
	client := New(Config{BaseURL: "http://127.0.0.1:1/?token=secret", CallerService: "zhicore-content"})

	err := client.DoJSON(context.Background(), http.MethodPost, "/users", "content.test", nil, nil)

	if err == nil {
		t.Fatal("DoJSON() error = nil, want transport error")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "token=") {
		t.Fatalf("error leaked sensitive URL: %v", err)
	}
}

func TestClientDoJSONPreservesContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()
	client := New(Config{BaseURL: server.URL, CallerService: "zhicore-content", HTTPClient: server.Client(), Timeout: time.Millisecond})

	err := client.DoJSON(context.Background(), http.MethodPost, "/slow", "content.test", nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout error = %v, want context deadline exceeded", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = client.DoJSON(ctx, http.MethodPost, "/slow", "content.test", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v, want context canceled", err)
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
