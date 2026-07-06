package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterRejectsStrictJSONDecodeErrorsBeforeServiceCall(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "trailing JSON value",
			body: `{"email":"user@example.com","nickname":"Alice","password":"Password123","emailVerificationToken":"verify-token"} {"extra":true}`,
		},
		{
			name: "unknown field",
			body: `{"email":"user@example.com","nickname":"Alice","password":"Password123","emailVerificationToken":"verify-token","extra":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeAuthService{}
			req := rawJSONRequest(http.MethodPost, "/api/v1/auth/register", tt.body)
			rr := httptest.NewRecorder()

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, http.StatusBadRequest, 1001)
			if service.registerCalled {
				t.Fatal("register service must not be called for invalid JSON framing or unknown fields")
			}
		})
	}
}

func TestLoginRejectsStrictJSONDecodeErrorsBeforeServiceCall(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "trailing JSON value",
			body: `{"email":"user@example.com","password":"Password123"} {"extra":true}`,
		},
		{
			name: "unknown field",
			body: `{"email":"user@example.com","password":"Password123","extra":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeAuthService{}
			req := rawJSONRequest(http.MethodPost, "/api/v1/auth/login", tt.body)
			rr := httptest.NewRecorder()

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, http.StatusBadRequest, 1001)
			if service.loginCalled {
				t.Fatal("login service must not be called for invalid JSON framing or unknown fields")
			}
		})
	}
}

func rawJSONRequest(method string, target string, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
