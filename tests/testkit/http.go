package testkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

type Envelope[T any] struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      T      `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

func DoJSON[T any](t *testing.T, client *http.Client, method string, url string, headers map[string]string, request any) Envelope[T] {
	t.Helper()

	var body bytes.Buffer
	if request != nil {
		if err := json.NewEncoder(&body).Encode(request); err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if request != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	var envelope Envelope[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode %s %s status=%d: %v", method, url, resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK || envelope.Code != 200 {
		t.Fatalf("%s %s status=%d envelope.code=%d message=%q", method, url, resp.StatusCode, envelope.Code, envelope.Message)
	}
	if envelope.Timestamp <= 0 {
		t.Fatalf("%s %s timestamp = %d, want positive", method, url, envelope.Timestamp)
	}
	return envelope
}

func RequireNonEmpty(t *testing.T, name string, value string) {
	t.Helper()
	if value == "" {
		t.Fatalf("%s must not be empty", name)
	}
}

func Path(baseURL string, path string) string {
	return fmt.Sprintf("%s%s", baseURL, path)
}
