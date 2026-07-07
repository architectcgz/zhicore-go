package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListSessionsUsesDefaultPaginationAndCurrentSession(t *testing.T) {
	lastSeenAt := time.Date(2026, 7, 4, 11, 0, 0, 0, time.UTC)
	service := &fakeAuthService{
		sessionsResult: ListSessionsResult{
			Items: []SessionSummary{
				{
					SessionID:   "session-current",
					CreatedAt:   time.Date(2026, 7, 3, 11, 0, 0, 0, time.UTC),
					LastSeenAt:  &lastSeenAt,
					ExpiresAt:   time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC),
					DeviceLabel: "Chrome on macOS",
					Current:     true,
				},
			},
			Page:  1,
			Size:  20,
			Total: 1,
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	withTrustedIdentityHeaders(req, "session-current")
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if service.listSessionsQuery.Page != 1 || service.listSessionsQuery.Size != 20 {
		t.Fatalf("pagination = (%d,%d), want (1,20)", service.listSessionsQuery.Page, service.listSessionsQuery.Size)
	}
	var body envelope[map[string]any]
	decodeJSON(t, rr.Body.Bytes(), &body)
	items := body.Data["items"].([]any)
	first := items[0].(map[string]any)
	if got := first["current"]; got != true {
		t.Fatalf("current = %v, want true", got)
	}
	if _, ok := first["lastAccessJti"]; ok {
		t.Fatal("lastAccessJti must not be present in session response")
	}
}

func TestListSessionsRequiresTrustedIdentityHeaders(t *testing.T) {
	service := &fakeAuthService{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
	if service.listSessionsCalled {
		t.Fatal("list sessions service must not be called without trusted identity headers")
	}
}

func TestRevokeSessionSuccess(t *testing.T) {
	service := &fakeAuthService{
		revokeSessionResult: RevokeSessionResult{
			SessionID: "session-other",
			Current:   false,
		},
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/session-other", nil)
	req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-1"})
	req.Header.Set(csrfHeaderName, "csrf-token-1")
	withTrustedIdentityHeaders(req, "session-current")
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var body envelope[map[string]any]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if got := body.Data["sessionId"]; got != "session-other" {
		t.Fatalf("sessionId = %v, want session-other", got)
	}
	if got := body.Data["current"]; got != false {
		t.Fatalf("current = %v, want false", got)
	}
}

func TestRevokeCurrentSessionAliasClearsCookies(t *testing.T) {
	service := &fakeAuthService{
		revokeSessionResult: RevokeSessionResult{
			SessionID: "session-current",
			Current:   true,
		},
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/current", nil)
	req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-1"})
	req.Header.Set(csrfHeaderName, "csrf-token-1")
	withTrustedIdentityHeaders(req, "session-current")
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	assertCookieCleared(t, rr.Result(), refreshTokenCookieName)
	assertCookieCleared(t, rr.Result(), csrfTokenCookieName)
}

func TestRevokeSessionReturnsAcceptedOrNotFound(t *testing.T) {
	t.Run("processing", func(t *testing.T) {
		service := &fakeAuthService{
			revokeSessionResult: RevokeSessionResult{
				SessionID: "session-other",
				Processing: &AcceptedSecurityOperation{
					OperationID:       "op-revoke",
					RetryAfterSeconds: 9,
				},
			},
		}
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/session-other", nil)
		req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-1"})
		req.Header.Set(csrfHeaderName, "csrf-token-1")
		withTrustedIdentityHeaders(req, "session-current")
		rr := httptest.NewRecorder()

		NewHandler(service).ServeHTTP(rr, req)

		assertAcceptedOperation(t, rr, "op-revoke")
		var body envelope[map[string]any]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if got := body.Data["sessionId"]; got != "session-other" {
			t.Fatalf("sessionId = %v, want session-other", got)
		}
		if got := int(body.Data["retryAfterSeconds"].(float64)); got != 9 {
			t.Fatalf("retryAfterSeconds = %d, want 9", got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		service := &fakeAuthService{revokeSessionErr: ErrDataNotFound}
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/session-other", nil)
		req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-1"})
		req.Header.Set(csrfHeaderName, "csrf-token-1")
		withTrustedIdentityHeaders(req, "session-current")
		rr := httptest.NewRecorder()

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusNotFound, 1005)
	})
}

func TestGetSecurityOperationSuccessAndNotFound(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		completedAt := time.Date(2026, 7, 4, 12, 2, 0, 0, time.UTC)
		retryAfter := 4
		service := &fakeAuthService{
			securityOperationResult: SecurityOperationResult{
				OperationID:       "op-123",
				Type:              "REVOKE_SESSION",
				Status:            "PROCESSING",
				CreatedAt:         time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC),
				UpdatedAt:         time.Date(2026, 7, 4, 12, 1, 0, 0, time.UTC),
				CompletedAt:       &completedAt,
				RetryAfterSeconds: &retryAfter,
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/security-operations/op-123", nil)
		withTrustedIdentityHeaders(req, "session-current")
		rr := httptest.NewRecorder()

		NewHandler(service).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		var body envelope[map[string]any]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if got := body.Data["operationId"]; got != "op-123" {
			t.Fatalf("operationId = %v, want op-123", got)
		}
		if _, ok := body.Data["redisKey"]; ok {
			t.Fatal("redisKey must not be present in security operation response")
		}
	})

	t.Run("not found", func(t *testing.T) {
		service := &fakeAuthService{securityOperationErr: ErrDataNotFound}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/security-operations/op-404", nil)
		withTrustedIdentityHeaders(req, "session-current")
		rr := httptest.NewRecorder()

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusNotFound, 1005)
	})
}
