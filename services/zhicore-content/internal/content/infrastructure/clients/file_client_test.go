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

	filecontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/file"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestFileClientValidateBodyMediaRefs(t *testing.T) {
	t.Run("validates refs through provider contract", func(t *testing.T) {
		var gotPath string
		var gotCallerService string
		var gotCallerOperation string
		var gotRequest filecontract.ValidateRefsRequest
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotCallerService = r.Header.Get("X-Caller-Service")
			gotCallerOperation = r.Header.Get("X-Caller-Operation")
			if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
				t.Fatalf("Decode request: %v", err)
			}
			writeSuccess(t, w, filecontract.ValidateRefsResponse{})
		}))
		defer server.Close()
		client := NewFileClient(FileClientConfig{BaseURL: server.URL, HTTPClient: server.Client(), MaxAttempts: 1})

		err := client.ValidateBodyMediaRefs(context.Background(), []ports.MediaRef{{FileID: "file_1"}})
		if err != nil {
			t.Fatalf("ValidateBodyMediaRefs() error = %v", err)
		}

		if gotPath != filecontract.ValidateRefsPath {
			t.Fatalf("path = %q, want %q", gotPath, filecontract.ValidateRefsPath)
		}
		if gotCallerService != "zhicore-content" || gotCallerOperation != filecontract.OperationContentValidateBodyMediaRefs {
			t.Fatalf("caller headers = %q/%q", gotCallerService, gotCallerOperation)
		}
		if gotRequest.Usage != filecontract.UsageContentBodyMedia || len(gotRequest.Refs) != 1 || gotRequest.Refs[0].FileID != "file_1" {
			t.Fatalf("request = %+v, want content body file_1", gotRequest)
		}
	})

	t.Run("maps invalid response and not found statuses to media semantic error", func(t *testing.T) {
		tests := []struct {
			name       string
			statusCode int
			body       any
		}{
			{name: "invalid body", statusCode: http.StatusOK, body: filecontract.ValidateRefsResponse{InvalidFileIDs: []string{"file_missing"}}},
			{name: "not found", statusCode: http.StatusNotFound, body: nil},
			{name: "gone", statusCode: http.StatusGone, body: nil},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tt.body == nil {
						w.WriteHeader(tt.statusCode)
						return
					}
					writeSuccess(t, w, tt.body)
				}))
				defer server.Close()
				client := NewFileClient(FileClientConfig{BaseURL: server.URL, HTTPClient: server.Client(), MaxAttempts: 1})

				err := client.ValidateBodyMediaRefs(context.Background(), []ports.MediaRef{{FileID: "file_missing"}})

				if !errors.Is(err, ports.ErrMediaRefInvalid) {
					t.Fatalf("error = %v, want ErrMediaRefInvalid", err)
				}
				if errors.Is(err, ports.ErrDependencyUnavailable) {
					t.Fatalf("error = %v, must not be dependency unavailable", err)
				}
			})
		}
	})

	t.Run("retries dependency failures without retrying semantic failures", func(t *testing.T) {
		var attempts int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		client := NewFileClient(FileClientConfig{BaseURL: server.URL, HTTPClient: server.Client(), MaxAttempts: 3})

		err := client.ValidateBodyMediaRefs(context.Background(), []ports.MediaRef{{FileID: "file_1"}})

		if !errors.Is(err, ports.ErrDependencyUnavailable) {
			t.Fatalf("error = %v, want ErrDependencyUnavailable", err)
		}
		if attempts != 3 {
			t.Fatalf("attempts = %d, want 3", attempts)
		}

		attempts = 0
		server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusNotFound)
		})

		err = client.ValidateBodyMediaRefs(context.Background(), []ports.MediaRef{{FileID: "file_missing"}})
		if !errors.Is(err, ports.ErrMediaRefInvalid) {
			t.Fatalf("error = %v, want ErrMediaRefInvalid", err)
		}
		if attempts != 1 {
			t.Fatalf("semantic attempts = %d, want 1", attempts)
		}
	})

	t.Run("maps timeout and cancellation without leaking base URL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
		}))
		defer server.Close()
		client := NewFileClient(FileClientConfig{
			BaseURL:     server.URL + "/?token=secret",
			HTTPClient:  server.Client(),
			Timeout:     time.Millisecond,
			MaxAttempts: 1,
		})

		err := client.ValidateBodyMediaRefs(context.Background(), []ports.MediaRef{{FileID: "file_1"}})
		if !errors.Is(err, ports.ErrDependencyUnavailable) {
			t.Fatalf("timeout error = %v, want ErrDependencyUnavailable", err)
		}
		assertNoSecretLeak(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err = client.ValidateBodyMediaRefs(ctx, []ports.MediaRef{{FileID: "file_1"}})
		if !errors.Is(err, ports.ErrDependencyUnavailable) || !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel error = %v, want dependency and context canceled", err)
		}
	})
}

func TestFileClientValidateCoverFile(t *testing.T) {
	t.Run("uses cover operation and maps invalid cover", func(t *testing.T) {
		var gotRequest filecontract.ValidateRefsRequest
		var gotCallerOperation string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotCallerOperation = r.Header.Get("X-Caller-Operation")
			if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
				t.Fatalf("Decode request: %v", err)
			}
			writeSuccess(t, w, filecontract.ValidateRefsResponse{InvalidFileIDs: []string{"cover_missing"}})
		}))
		defer server.Close()
		client := NewFileClient(FileClientConfig{BaseURL: server.URL, HTTPClient: server.Client(), MaxAttempts: 1})

		err := client.ValidateCoverFile(context.Background(), "cover_missing")

		if !errors.Is(err, ports.ErrCoverUnavailable) {
			t.Fatalf("error = %v, want ErrCoverUnavailable", err)
		}
		if gotCallerOperation != filecontract.OperationContentValidateCover {
			t.Fatalf("caller operation = %q, want cover operation", gotCallerOperation)
		}
		if gotRequest.Usage != filecontract.UsageContentCover || len(gotRequest.Refs) != 1 || gotRequest.Refs[0].FileID != "cover_missing" {
			t.Fatalf("request = %+v, want cover reference", gotRequest)
		}
	})

	t.Run("maps cover not found and dependency failure", func(t *testing.T) {
		tests := []struct {
			statusCode int
			want       error
		}{
			{statusCode: http.StatusNotFound, want: ports.ErrCoverUnavailable},
			{statusCode: http.StatusGone, want: ports.ErrCoverUnavailable},
			{statusCode: http.StatusInternalServerError, want: ports.ErrDependencyUnavailable},
		}
		for _, tt := range tests {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			client := NewFileClient(FileClientConfig{BaseURL: server.URL, HTTPClient: server.Client(), MaxAttempts: 1})
			err := client.ValidateCoverFile(context.Background(), "cover_1")
			server.Close()

			if !errors.Is(err, tt.want) {
				t.Fatalf("status %d error = %v, want %v", tt.statusCode, err, tt.want)
			}
		}
	})
}

func TestFileClientRejectsBlankReferencesLocally(t *testing.T) {
	client := NewFileClient(FileClientConfig{BaseURL: "http://127.0.0.1", HTTPClient: http.DefaultClient})

	if err := client.ValidateBodyMediaRefs(context.Background(), []ports.MediaRef{{FileID: "  "}}); !errors.Is(err, ports.ErrMediaRefInvalid) {
		t.Fatalf("blank media error = %v, want ErrMediaRefInvalid", err)
	}
	if err := client.ValidateCoverFile(context.Background(), " "); !errors.Is(err, ports.ErrCoverUnavailable) {
		t.Fatalf("blank cover error = %v, want ErrCoverUnavailable", err)
	}
}

func assertNoRawURLLeak(t *testing.T, err error) {
	t.Helper()
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "token=") {
		t.Fatalf("error leaked sensitive URL: %v", err)
	}
}
