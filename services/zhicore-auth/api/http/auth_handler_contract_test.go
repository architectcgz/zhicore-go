package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/domain"
)

func TestRefreshCoreErrorContracts(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "token invalid", err: ErrTokenInvalid, wantStatus: http.StatusUnauthorized, wantCode: 2001},
		{name: "token expired", err: ErrTokenExpired, wantStatus: http.StatusUnauthorized, wantCode: 2002},
		{name: "token replayed", err: ErrTokenReplayed, wantStatus: http.StatusUnauthorized, wantCode: 2017},
		{name: "session revoked", err: ErrSessionRevoked, wantStatus: http.StatusUnauthorized, wantCode: 2018},
		{name: "account disabled", err: domain.ErrAccountDisabled, wantStatus: http.StatusForbidden, wantCode: 2004},
		{name: "account banned", err: domain.ErrAccountBanned, wantStatus: http.StatusForbidden, wantCode: 2019},
		{name: "account locked", err: domain.ErrAccountLocked, wantStatus: http.StatusForbidden, wantCode: 2014},
		{name: "rate limited", err: domain.ErrRateLimitExceeded, wantStatus: http.StatusTooManyRequests, wantCode: 2015},
		{name: "service degraded", err: domain.ErrRateLimitUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 1004},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeAuthService{refreshErr: tc.err}
			req := newRefreshRequest()
			rr := httptest.NewRecorder()

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestLogoutCoreContracts(t *testing.T) {
	t.Run("no credentials clears cookies locally without calling service", func(t *testing.T) {
		service := &fakeAuthService{logoutErr: errors.New("logout service must not be called")}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		rr := httptest.NewRecorder()

		NewHandler(service).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		var body envelope[map[string]any]
		decodeJSON(t, rr.Body.Bytes(), &body)
		if body.Code != 200 {
			t.Fatalf("body code = %d, want 200; body=%s", body.Code, rr.Body.String())
		}
		if got := body.Data["loggedOut"]; got != true {
			t.Fatalf("loggedOut = %v, want true", got)
		}
		if got := body.Data["serverRevoked"]; got != false {
			t.Fatalf("serverRevoked = %v, want false", got)
		}
		assertCookieCleared(t, rr.Result(), refreshTokenCookieName)
		assertCookieCleared(t, rr.Result(), csrfTokenCookieName)
		if service.logoutCalled {
			t.Fatal("logout service must not be called when request has no trusted identity or refresh cookie")
		}
	})

	t.Run("trusted identity without refresh cookie still requires csrf", func(t *testing.T) {
		service := &fakeAuthService{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		withTrustedIdentityHeaders(req, "session-logout")
		rr := httptest.NewRecorder()

		NewHandler(service).ServeHTTP(rr, req)

		assertErrorEnvelope(t, rr, http.StatusForbidden, 2013)
		if service.logoutCalled {
			t.Fatal("logout service must not be called when trusted identity revoke skips csrf")
		}
	})

	t.Run("processing uses success envelope", func(t *testing.T) {
		service := &fakeAuthService{
			logoutResult: LogoutResult{
				Processing: &AcceptedSecurityOperation{
					OperationID:       "op-logout",
					RetryAfterSeconds: 7,
				},
			},
		}
		req := newLogoutRequest(true)
		rr := httptest.NewRecorder()

		NewHandler(service).ServeHTTP(rr, req)

		assertAcceptedOperation(t, rr, "op-logout")
	})

	testCases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "session revoked", err: ErrSessionRevoked, wantStatus: http.StatusUnauthorized, wantCode: 2018},
		{name: "rate limited", err: domain.ErrRateLimitExceeded, wantStatus: http.StatusTooManyRequests, wantCode: 2015},
		{name: "service degraded", err: domain.ErrRateLimitUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 1004},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeAuthService{logoutErr: tc.err}
			req := newLogoutRequest(true)
			rr := httptest.NewRecorder()

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestRevokeCurrentSessionCoreContracts(t *testing.T) {
	t.Run("processing uses success envelope", func(t *testing.T) {
		service := &fakeAuthService{
			revokeSessionResult: RevokeSessionResult{
				SessionID: "session-current",
				Current:   true,
				Processing: &AcceptedSecurityOperation{
					OperationID:       "op-revoke-current",
					RetryAfterSeconds: 11,
				},
			},
		}
		req := newRevokeCurrentRequest(true)
		rr := httptest.NewRecorder()

		NewHandler(service).ServeHTTP(rr, req)

		assertAcceptedOperation(t, rr, "op-revoke-current")
	})

	testCases := []struct {
		name       string
		req        *http.Request
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "login required", req: httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/current", nil), wantStatus: http.StatusUnauthorized, wantCode: 2006},
		{name: "csrf invalid", req: newRevokeCurrentRequest(false), wantStatus: http.StatusForbidden, wantCode: 2013},
		{name: "session revoked", req: newRevokeCurrentRequest(true), err: ErrSessionRevoked, wantStatus: http.StatusUnauthorized, wantCode: 2018},
		{name: "rate limited", req: newRevokeCurrentRequest(true), err: domain.ErrRateLimitExceeded, wantStatus: http.StatusTooManyRequests, wantCode: 2015},
		{name: "service degraded", req: newRevokeCurrentRequest(true), err: domain.ErrRateLimitUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 1004},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeAuthService{revokeSessionErr: tc.err}
			rr := httptest.NewRecorder()

			NewHandler(service).ServeHTTP(rr, tc.req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestMeCoreErrorContracts(t *testing.T) {
	testCases := []struct {
		name       string
		req        *http.Request
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "login required", req: httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil), wantStatus: http.StatusUnauthorized, wantCode: 2006},
		{name: "token invalid", req: newMeRequest(), err: ErrTokenInvalid, wantStatus: http.StatusUnauthorized, wantCode: 2001},
		{name: "token expired", req: newMeRequest(), err: ErrTokenExpired, wantStatus: http.StatusUnauthorized, wantCode: 2002},
		{name: "session revoked", req: newMeRequest(), err: ErrSessionRevoked, wantStatus: http.StatusUnauthorized, wantCode: 2018},
		{name: "account disabled", req: newMeRequest(), err: domain.ErrAccountDisabled, wantStatus: http.StatusForbidden, wantCode: 2004},
		{name: "account banned", req: newMeRequest(), err: domain.ErrAccountBanned, wantStatus: http.StatusForbidden, wantCode: 2019},
		{name: "rate limited", req: newMeRequest(), err: domain.ErrRateLimitExceeded, wantStatus: http.StatusTooManyRequests, wantCode: 2015},
		{name: "principal unavailable", req: newMeRequest(), err: ErrPrincipalUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 2016},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeAuthService{meErr: tc.err}
			rr := httptest.NewRecorder()

			NewHandler(service).ServeHTTP(rr, tc.req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestOtherEndpointCoreErrorContracts(t *testing.T) {
	t.Run("revoke session errors", func(t *testing.T) {
		testCases := []struct {
			name       string
			req        *http.Request
			err        error
			wantStatus int
			wantCode   int
		}{
			{name: "validation", req: newRevokeSessionRequest("", true), wantStatus: http.StatusBadRequest, wantCode: 1001},
			{name: "login required", req: httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/session-other", nil), wantStatus: http.StatusUnauthorized, wantCode: 2006},
			{name: "csrf invalid", req: newRevokeSessionRequest("session-other", false), wantStatus: http.StatusForbidden, wantCode: 2013},
			{name: "not found", req: newRevokeSessionRequest("session-other", true), err: ErrDataNotFound, wantStatus: http.StatusNotFound, wantCode: 1005},
			{name: "rate limited", req: newRevokeSessionRequest("session-other", true), err: domain.ErrRateLimitExceeded, wantStatus: http.StatusTooManyRequests, wantCode: 2015},
			{name: "service degraded", req: newRevokeSessionRequest("session-other", true), err: domain.ErrRateLimitUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 1004},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				service := &fakeAuthService{revokeSessionErr: tc.err}
				rr := httptest.NewRecorder()

				NewHandler(service).ServeHTTP(rr, tc.req)

				assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
			})
		}
	})

	t.Run("csrf endpoint errors", func(t *testing.T) {
		testCases := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   int
		}{
			{name: "rate limited", err: domain.ErrRateLimitExceeded, wantStatus: http.StatusTooManyRequests, wantCode: 2015},
			{name: "service degraded", err: domain.ErrRateLimitUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 1004},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				service := &fakeAuthService{csrfErr: tc.err}
				req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/csrf", nil)
				rr := httptest.NewRecorder()

				NewHandler(service).ServeHTTP(rr, req)

				assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
			})
		}
	})

	t.Run("list sessions errors", func(t *testing.T) {
		testCases := []struct {
			name       string
			req        *http.Request
			err        error
			wantStatus int
			wantCode   int
		}{
			{name: "validation", req: newListSessionsRequest("?page=0"), wantStatus: http.StatusBadRequest, wantCode: 1001},
			{name: "login required", req: httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil), wantStatus: http.StatusUnauthorized, wantCode: 2006},
			{name: "rate limited", req: newListSessionsRequest(""), err: domain.ErrRateLimitExceeded, wantStatus: http.StatusTooManyRequests, wantCode: 2015},
			{name: "service degraded", req: newListSessionsRequest(""), err: domain.ErrRateLimitUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 1004},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				service := &fakeAuthService{sessionsErr: tc.err}
				rr := httptest.NewRecorder()

				NewHandler(service).ServeHTTP(rr, tc.req)

				assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
			})
		}
	})

	t.Run("security operation errors", func(t *testing.T) {
		testCases := []struct {
			name       string
			req        *http.Request
			err        error
			wantStatus int
			wantCode   int
		}{
			{name: "validation", req: newSecurityOperationRequest(""), wantStatus: http.StatusBadRequest, wantCode: 1001},
			{name: "login required", req: httptest.NewRequest(http.MethodGet, "/api/v1/auth/security-operations/op-1", nil), wantStatus: http.StatusUnauthorized, wantCode: 2006},
			{name: "not found", req: newSecurityOperationRequest("op-404"), err: ErrDataNotFound, wantStatus: http.StatusNotFound, wantCode: 1005},
			{name: "permission denied", req: newSecurityOperationRequest("op-403"), err: ErrPermissionDenied, wantStatus: http.StatusForbidden, wantCode: 2005},
			{name: "rate limited", req: newSecurityOperationRequest("op-429"), err: domain.ErrRateLimitExceeded, wantStatus: http.StatusTooManyRequests, wantCode: 2015},
			{name: "service degraded", req: newSecurityOperationRequest("op-503"), err: domain.ErrRateLimitUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 1004},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				service := &fakeAuthService{securityOperationErr: tc.err}
				rr := httptest.NewRecorder()

				NewHandler(service).ServeHTTP(rr, tc.req)

				assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
			})
		}
	})
}

func assertAcceptedOperation(t *testing.T, rr *httptest.ResponseRecorder, wantOperationID string) {
	t.Helper()
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusAccepted, rr.Body.String())
	}
	var body envelope[map[string]any]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if body.Code != 200 {
		t.Fatalf("body code = %d, want 200; body=%s", body.Code, rr.Body.String())
	}
	if got := body.Data["operationId"]; got != wantOperationID {
		t.Fatalf("operationId = %v, want %s", got, wantOperationID)
	}
	if got := body.Data["status"]; got != "PROCESSING" {
		t.Fatalf("status = %v, want PROCESSING; body=%s", got, rr.Body.String())
	}
}

func newRefreshRequest() *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "refresh-token-1"})
	req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-1"})
	req.Header.Set(csrfHeaderName, "csrf-token-1")
	return req
}

func newLogoutRequest(withRefresh bool) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	withTrustedIdentityHeaders(req, "session-logout")
	if withRefresh {
		req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "refresh-token-1"})
		req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-1"})
		req.Header.Set(csrfHeaderName, "csrf-token-1")
	}
	return req
}

func newRevokeCurrentRequest(withValidCSRF bool) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/current", nil)
	withTrustedIdentityHeaders(req, "session-current")
	req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-1"})
	if withValidCSRF {
		req.Header.Set(csrfHeaderName, "csrf-token-1")
	} else {
		req.Header.Set(csrfHeaderName, "csrf-token-other")
	}
	return req
}

func newRevokeSessionRequest(sessionID string, withValidCSRF bool) *http.Request {
	targetSessionID := sessionID
	if targetSessionID == "" {
		targetSessionID = "%20"
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+targetSessionID, nil)
	withTrustedIdentityHeaders(req, "session-current")
	req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-1"})
	if withValidCSRF {
		req.Header.Set(csrfHeaderName, "csrf-token-1")
	} else {
		req.Header.Set(csrfHeaderName, "csrf-token-other")
	}
	return req
}

func newMeRequest() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	withTrustedIdentityHeaders(req, "session-me")
	return req
}

func newListSessionsRequest(query string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions"+query, nil)
	withTrustedIdentityHeaders(req, "session-current")
	return req
}

func newSecurityOperationRequest(operationID string) *http.Request {
	targetOperationID := operationID
	if targetOperationID == "" {
		targetOperationID = "%20"
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/security-operations/"+targetOperationID, nil)
	withTrustedIdentityHeaders(req, "session-current")
	return req
}
