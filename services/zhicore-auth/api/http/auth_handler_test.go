package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/domain"
)

func TestRegisterSuccessSetsCookiesAndReturnsAuthenticatedPrincipal(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	service := &fakeAuthService{
		registerResult: RegisterResult{
			Registered:            true,
			Authenticated:         true,
			AccessToken:           "access-token",
			AccessTokenExpiresAt:  now.Add(2 * time.Hour),
			RefreshToken:          "refresh-token",
			RefreshTokenExpiresAt: now.Add(30 * 24 * time.Hour),
			CSRFToken:             "csrf-token",
			Principal:             authPrincipalFixture("session-current"),
		},
	}

	req := jsonRequest(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email":                  "user@example.com",
		"nickname":               "Alice",
		"password":               "Password123",
		"emailVerificationToken": "verify-token",
	})
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var body envelope[map[string]any]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if body.Code != 200 {
		t.Fatalf("body code = %d, want 200", body.Code)
	}
	if got := body.Data["registered"]; got != true {
		t.Fatalf("registered = %v, want true", got)
	}
	if got := body.Data["authenticated"]; got != true {
		t.Fatalf("authenticated = %v, want true", got)
	}
	if got := body.Data["accessToken"]; got != "access-token" {
		t.Fatalf("accessToken = %v, want access-token", got)
	}
	if got := body.Data["tokenType"]; got != "Bearer" {
		t.Fatalf("tokenType = %v, want Bearer", got)
	}
	if got := int(body.Data["expiresIn"].(float64)); got != 7200 {
		t.Fatalf("expiresIn = %d, want 7200", got)
	}
	if got := body.Data["csrfToken"]; got != "csrf-token" {
		t.Fatalf("csrfToken = %v, want csrf-token", got)
	}
	if _, ok := body.Data["refreshToken"]; ok {
		t.Fatal("refreshToken must not be present in response body")
	}
	refreshCookie := requireCookie(t, rr.Result(), refreshTokenCookieName)
	if refreshCookie.Value != "refresh-token" {
		t.Fatalf("refresh cookie value = %q, want refresh-token", refreshCookie.Value)
	}
	if !refreshCookie.HttpOnly {
		t.Fatal("refresh cookie must be HttpOnly")
	}
	if refreshCookie.Path != authCookiePath {
		t.Fatalf("refresh cookie path = %q, want %q", refreshCookie.Path, authCookiePath)
	}
	csrfCookie := requireCookie(t, rr.Result(), csrfTokenCookieName)
	if csrfCookie.Value != "csrf-token" {
		t.Fatalf("csrf cookie value = %q, want csrf-token", csrfCookie.Value)
	}
	if csrfCookie.HttpOnly {
		t.Fatal("csrf cookie must not be HttpOnly")
	}
	if service.registerInput.EmailVerificationToken != "verify-token" {
		t.Fatalf("emailVerificationToken = %q, want verify-token", service.registerInput.EmailVerificationToken)
	}
}

func TestRegisterMapsCoreErrors(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "invalid email", err: domain.ErrEmailInvalid, wantStatus: http.StatusBadRequest, wantCode: 2010},
		{name: "invalid password", err: ErrPasswordInvalid, wantStatus: http.StatusBadRequest, wantCode: 2011},
		{name: "email exists", err: ErrEmailExists, wantStatus: http.StatusConflict, wantCode: 2009},
		{name: "pending retryable", err: ErrRegisterPendingRetryable, wantStatus: http.StatusServiceUnavailable, wantCode: 2012},
		{name: "rate limited", err: domain.ErrRateLimitExceeded, wantStatus: http.StatusTooManyRequests, wantCode: 2015},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeAuthService{registerErr: tc.err}
			req := jsonRequest(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
				"email":                  "user@example.com",
				"nickname":               "Alice",
				"password":               "Password123",
				"emailVerificationToken": "verify-token",
			})
			rr := httptest.NewRecorder()

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestLoginSuccessSetsCookiesAndOmitsRefreshTokenFromBody(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	service := &fakeAuthService{
		loginResult: LoginResult{
			AccessToken:           "access-token",
			AccessTokenExpiresAt:  now.Add(2 * time.Hour),
			RefreshToken:          "refresh-token",
			RefreshTokenExpiresAt: now.Add(30 * 24 * time.Hour),
			CSRFToken:             "csrf-token",
			Principal:             authPrincipalFixture("session-login"),
		},
	}

	req := jsonRequest(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "user@example.com",
		"password": "Password123",
	})
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var body envelope[map[string]any]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if _, ok := body.Data["refreshToken"]; ok {
		t.Fatal("refreshToken must not be present in response body")
	}
	requireCookie(t, rr.Result(), refreshTokenCookieName)
	requireCookie(t, rr.Result(), csrfTokenCookieName)
	if service.loginInput.Email != "user@example.com" {
		t.Fatalf("login email = %q, want user@example.com", service.loginInput.Email)
	}
}

func TestLoginMapsCoreErrors(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{name: "invalid credentials", err: domain.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized, wantCode: 2003},
		{name: "disabled", err: domain.ErrAccountDisabled, wantStatus: http.StatusForbidden, wantCode: 2004},
		{name: "banned", err: domain.ErrAccountBanned, wantStatus: http.StatusForbidden, wantCode: 2019},
		{name: "locked", err: domain.ErrAccountLocked, wantStatus: http.StatusForbidden, wantCode: 2014},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeAuthService{loginErr: tc.err}
			req := jsonRequest(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
				"email":    "user@example.com",
				"password": "Password123",
			})
			rr := httptest.NewRecorder()

			NewHandler(service).ServeHTTP(rr, req)

			assertErrorEnvelope(t, rr, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestRefreshSuccessRotatesCookiesAndReturnsPrincipal(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	service := &fakeAuthService{
		refreshResult: RefreshResult{
			AccessToken:           "access-token-2",
			AccessTokenExpiresAt:  now.Add(2 * time.Hour),
			RefreshToken:          "refresh-token-2",
			RefreshTokenExpiresAt: now.Add(30 * 24 * time.Hour),
			CSRFToken:             "csrf-token-2",
			Principal:             authPrincipalFixture("session-refresh"),
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "refresh-token-1"})
	req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-1"})
	req.Header.Set(csrfHeaderName, "csrf-token-1")
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	requireCookie(t, rr.Result(), refreshTokenCookieName)
	requireCookie(t, rr.Result(), csrfTokenCookieName)
	if service.refreshInput.RefreshToken != "refresh-token-1" {
		t.Fatalf("refresh token = %q, want refresh-token-1", service.refreshInput.RefreshToken)
	}
}

func TestRefreshReturnsAcceptedWhenProjectionIsStillProcessing(t *testing.T) {
	service := &fakeAuthService{
		refreshResult: RefreshResult{
			Processing: &AcceptedSecurityOperation{
				OperationID:       "op-refresh",
				RetryAfterSeconds: 5,
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "refresh-token-1"})
	req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-1"})
	req.Header.Set(csrfHeaderName, "csrf-token-1")
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	assertAcceptedOperation(t, rr, "op-refresh")
	var body envelope[map[string]any]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if got := int(body.Data["retryAfterSeconds"].(float64)); got != 5 {
		t.Fatalf("retryAfterSeconds = %d, want 5", got)
	}
	if got := body.Data["refreshAccepted"]; got != false {
		t.Fatalf("refreshAccepted = %v, want false", got)
	}
}

func TestRefreshRejectsInvalidCSRFBeforeCallingService(t *testing.T) {
	service := &fakeAuthService{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "refresh-token-1"})
	req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-cookie"})
	req.Header.Set(csrfHeaderName, "csrf-token-header")
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusForbidden, 2013)
	if service.refreshCalled {
		t.Fatal("refresh service must not be called when csrf validation fails")
	}
}

func TestLogoutSuccessClearsCookiesAndReturnsServerRevoked(t *testing.T) {
	service := &fakeAuthService{
		logoutResult: LogoutResult{
			LoggedOut:     true,
			ServerRevoked: true,
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "refresh-token-1"})
	req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-1"})
	req.Header.Set(csrfHeaderName, "csrf-token-1")
	withTrustedIdentityHeaders(req, "session-logout")
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var body envelope[map[string]any]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if got := body.Data["loggedOut"]; got != true {
		t.Fatalf("loggedOut = %v, want true", got)
	}
	if got := body.Data["serverRevoked"]; got != true {
		t.Fatalf("serverRevoked = %v, want true", got)
	}
	assertCookieCleared(t, rr.Result(), refreshTokenCookieName)
	assertCookieCleared(t, rr.Result(), csrfTokenCookieName)
}

func TestLogoutRejectsInvalidCSRFWhenRefreshCookiePresent(t *testing.T) {
	service := &fakeAuthService{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "refresh-token-1"})
	req.AddCookie(&http.Cookie{Name: csrfTokenCookieName, Value: "csrf-token-cookie"})
	req.Header.Set(csrfHeaderName, "csrf-token-header")
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusForbidden, 2013)
	if service.logoutCalled {
		t.Fatal("logout service must not be called when csrf validation fails")
	}
}

func TestGetCurrentPrincipalSuccess(t *testing.T) {
	service := &fakeAuthService{
		currentPrincipal: Principal{
			AccountID:        "1001",
			UserID:           "2001",
			Email:            "user@example.com",
			Roles:            []string{"ROLE_USER"},
			AccountStatus:    "ACTIVE",
			SessionID:        "session-me",
			SessionVersion:   3,
			PrincipalVersion: 4,
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	withTrustedIdentityHeaders(req, "session-me")
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var body envelope[map[string]map[string]any]
	decodeJSON(t, rr.Body.Bytes(), &body)
	principal := body.Data["principal"]
	if got := principal["sessionId"]; got != "session-me" {
		t.Fatalf("sessionId = %v, want session-me", got)
	}
	if _, ok := principal["nickname"]; ok {
		t.Fatal("nickname must not be present in auth/me response")
	}
}

func TestGetCurrentPrincipalRequiresTrustedIdentityHeaders(t *testing.T) {
	service := &fakeAuthService{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusUnauthorized, 2006)
	if service.meCalled {
		t.Fatal("me service must not be called without trusted identity headers")
	}
}

func TestGetCSRFTokenSetsCookie(t *testing.T) {
	service := &fakeAuthService{
		csrfResult: IssueCSRFTokenResult{CSRFToken: "csrf-issued"},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/csrf", nil)
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var body envelope[map[string]any]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if got := body.Data["csrfToken"]; got != "csrf-issued" {
		t.Fatalf("csrfToken = %v, want csrf-issued", got)
	}
	csrfCookie := requireCookie(t, rr.Result(), csrfTokenCookieName)
	if csrfCookie.Value != "csrf-issued" {
		t.Fatalf("csrf cookie = %q, want csrf-issued", csrfCookie.Value)
	}
	if csrfCookie.HttpOnly {
		t.Fatal("csrf token cookie must not be HttpOnly")
	}
}

func TestGetCSRFTokenMapsRateLimited(t *testing.T) {
	service := &fakeAuthService{csrfErr: domain.ErrRateLimitExceeded}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/csrf", nil)
	rr := httptest.NewRecorder()

	NewHandler(service).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusTooManyRequests, 2015)
}

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

type fakeAuthService struct {
	registerInput  RegisterCommand
	registerResult RegisterResult
	registerErr    error

	loginInput  LoginCommand
	loginResult LoginResult
	loginErr    error

	refreshCalled bool
	refreshInput  RefreshCommand
	refreshResult RefreshResult
	refreshErr    error

	logoutCalled bool
	logoutInput  LogoutCommand
	logoutResult LogoutResult
	logoutErr    error

	meCalled         bool
	currentPrincipal Principal
	meErr            error

	csrfResult IssueCSRFTokenResult
	csrfErr    error

	listSessionsCalled bool
	listSessionsQuery  ListSessionsQuery
	sessionsResult     ListSessionsResult
	sessionsErr        error

	revokeSessionInput  RevokeSessionCommand
	revokeSessionResult RevokeSessionResult
	revokeSessionErr    error

	securityOperationQuery  GetSecurityOperationQuery
	securityOperationResult SecurityOperationResult
	securityOperationErr    error
}

func (f *fakeAuthService) Register(ctx context.Context, cmd RegisterCommand) (RegisterResult, error) {
	f.registerInput = cmd
	return f.registerResult, normalizeTestServiceError(f.registerErr)
}

func (f *fakeAuthService) Login(ctx context.Context, cmd LoginCommand) (LoginResult, error) {
	f.loginInput = cmd
	return f.loginResult, normalizeTestServiceError(f.loginErr)
}

func (f *fakeAuthService) Refresh(ctx context.Context, cmd RefreshCommand) (RefreshResult, error) {
	f.refreshCalled = true
	f.refreshInput = cmd
	return f.refreshResult, normalizeTestServiceError(f.refreshErr)
}

func (f *fakeAuthService) Logout(ctx context.Context, cmd LogoutCommand) (LogoutResult, error) {
	f.logoutCalled = true
	f.logoutInput = cmd
	return f.logoutResult, normalizeTestServiceError(f.logoutErr)
}

func (f *fakeAuthService) GetCurrentPrincipal(ctx context.Context, query CurrentPrincipalQuery) (Principal, error) {
	f.meCalled = true
	return f.currentPrincipal, normalizeTestServiceError(f.meErr)
}

func (f *fakeAuthService) IssueCSRFToken(ctx context.Context) (IssueCSRFTokenResult, error) {
	return f.csrfResult, normalizeTestServiceError(f.csrfErr)
}

func (f *fakeAuthService) ListSessions(ctx context.Context, query ListSessionsQuery) (ListSessionsResult, error) {
	f.listSessionsCalled = true
	f.listSessionsQuery = query
	return f.sessionsResult, normalizeTestServiceError(f.sessionsErr)
}

func (f *fakeAuthService) RevokeSession(ctx context.Context, cmd RevokeSessionCommand) (RevokeSessionResult, error) {
	f.revokeSessionInput = cmd
	return f.revokeSessionResult, normalizeTestServiceError(f.revokeSessionErr)
}

func (f *fakeAuthService) GetSecurityOperation(ctx context.Context, query GetSecurityOperationQuery) (SecurityOperationResult, error) {
	f.securityOperationQuery = query
	return f.securityOperationResult, normalizeTestServiceError(f.securityOperationErr)
}

func normalizeTestServiceError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrEmailInvalid):
		return ErrEmailInvalid
	case errors.Is(err, domain.ErrInvalidCredentials):
		return ErrInvalidCredentials
	case errors.Is(err, domain.ErrAccountDisabled):
		return ErrAccountDisabled
	case errors.Is(err, domain.ErrAccountBanned):
		return ErrAccountBanned
	case errors.Is(err, domain.ErrAccountLocked):
		return ErrAccountLocked
	case errors.Is(err, domain.ErrRateLimitExceeded):
		return ErrRateLimited
	case errors.Is(err, domain.ErrRateLimitUnavailable):
		return ErrServiceDegraded
	default:
		return err
	}
}

type envelope[T any] struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      T      `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

func jsonRequest(t *testing.T, method string, target string, payload any) *http.Request {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decodeJSON(t *testing.T, payload []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body=%s", err, string(payload))
	}
}

func assertErrorEnvelope(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int, wantCode int) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, wantStatus, rr.Body.String())
	}
	var body envelope[map[string]any]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if body.Code != wantCode {
		t.Fatalf("body code = %d, want %d; body=%s", body.Code, wantCode, rr.Body.String())
	}
}

func requireCookie(t *testing.T, res *http.Response, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range res.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found", name)
	return nil
}

func assertCookieCleared(t *testing.T, res *http.Response, name string) {
	t.Helper()
	cookie := requireCookie(t, res, name)
	if cookie.MaxAge >= 0 {
		t.Fatalf("cookie %q maxAge = %d, want negative for clear", name, cookie.MaxAge)
	}
}

func withTrustedIdentityHeaders(req *http.Request, sessionID string) {
	req.Header.Set(accountIDHeaderName, "1001")
	req.Header.Set(userIDHeaderName, "2001")
	req.Header.Set(sessionIDHeaderName, sessionID)
	req.Header.Set(sessionVersionHeaderName, "3")
	req.Header.Set(principalVersionHeaderName, "4")
	req.Header.Set(userRolesHeaderName, "ROLE_USER,ROLE_MODERATOR")
}

func authPrincipalFixture(sessionID string) Principal {
	return Principal{
		AccountID:        "1001",
		UserID:           "2001",
		Email:            "user@example.com",
		Roles:            []string{"ROLE_USER", "ROLE_MODERATOR"},
		AccountStatus:    "ACTIVE",
		SessionID:        sessionID,
		SessionVersion:   3,
		PrincipalVersion: 4,
	}
}

var (
	_ error = ErrPasswordInvalid
	_ error = ErrEmailExists
	_ error = ErrRegisterPendingRetryable
	_ error = ErrDataNotFound
	_ error = errors.New("")
)
