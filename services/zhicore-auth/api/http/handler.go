package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/gin-gonic/gin"
)

const (
	refreshTokenCookieName     = "refresh_token"
	csrfTokenCookieName        = "csrf_token"
	csrfHeaderName             = "X-CSRF-Token"
	authCookiePath             = "/api/v1/auth"
	accountIDHeaderName        = "X-Account-Id"
	userIDHeaderName           = "X-User-Id"
	userRolesHeaderName        = "X-User-Roles"
	sessionIDHeaderName        = "X-Session-Id"
	sessionVersionHeaderName   = "X-Session-Version"
	principalVersionHeaderName = "X-Principal-Version"

	defaultPageSize = 20
	maxPageSize     = 50
	fixedExpiresIn  = 7200
)

var (
	ErrEmailInvalid             = errors.New("email is invalid")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrAccountDisabled          = errors.New("account is disabled")
	ErrAccountBanned            = errors.New("account is banned")
	ErrAccountLocked            = errors.New("account is locked")
	ErrRateLimited              = errors.New("rate limit exceeded")
	ErrServiceDegraded          = errors.New("service degraded")
	ErrPasswordInvalid          = errors.New("password is invalid")
	ErrEmailExists              = errors.New("email already exists")
	ErrRegisterPendingRetryable = errors.New("register pending retryable")
	ErrTokenInvalid             = errors.New("token is invalid")
	ErrTokenExpired             = errors.New("token is expired")
	ErrTokenReplayed            = errors.New("token is replayed")
	ErrSessionRevoked           = errors.New("session is revoked")
	ErrCSRFInvalid              = errors.New("csrf is invalid")
	ErrLoginRequired            = errors.New("login required")
	ErrPermissionDenied         = errors.New("permission denied")
	ErrRoleRequired             = errors.New("role required")
	ErrResourceAccessDenied     = errors.New("resource access denied")
	ErrPrincipalUnavailable     = errors.New("principal unavailable")
	ErrDataNotFound             = errors.New("data not found")
)

type Service interface {
	Register(ctx context.Context, cmd RegisterCommand) (RegisterResult, error)
	Login(ctx context.Context, cmd LoginCommand) (LoginResult, error)
	Refresh(ctx context.Context, cmd RefreshCommand) (RefreshResult, error)
	Logout(ctx context.Context, cmd LogoutCommand) (LogoutResult, error)
	GetCurrentPrincipal(ctx context.Context, query CurrentPrincipalQuery) (Principal, error)
	IssueCSRFToken(ctx context.Context) (IssueCSRFTokenResult, error)
	ListSessions(ctx context.Context, query ListSessionsQuery) (ListSessionsResult, error)
	RevokeSession(ctx context.Context, cmd RevokeSessionCommand) (RevokeSessionResult, error)
	GetSecurityOperation(ctx context.Context, query GetSecurityOperationQuery) (SecurityOperationResult, error)
}

type RegisterCommand struct {
	Email                  string
	Nickname               string
	Password               string
	EmailVerificationToken string
}

type RegisterResult struct {
	Registered            bool
	Authenticated         bool
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	CSRFToken             string
	Principal             Principal
	LoginDeferredReason   string
}

type LoginCommand struct {
	Email    string
	Password string
}

type LoginResult struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	CSRFToken             string
	Principal             Principal
}

type RefreshCommand struct {
	RefreshToken string
	CSRFToken    string
}

type RefreshResult struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	CSRFToken             string
	Principal             Principal
	Processing            *AcceptedSecurityOperation
}

type LogoutCommand struct {
	Identity     TrustedIdentity
	HasIdentity  bool
	RefreshToken string
	CSRFToken    string
}

type LogoutResult struct {
	LoggedOut     bool
	ServerRevoked bool
	Processing    *AcceptedSecurityOperation
}

type CurrentPrincipalQuery struct {
	Identity TrustedIdentity
}

type IssueCSRFTokenResult struct {
	CSRFToken string
}

type ListSessionsQuery struct {
	Identity TrustedIdentity
	Page     int
	Size     int
}

type SessionSummary struct {
	SessionID   string
	CreatedAt   time.Time
	LastSeenAt  *time.Time
	ExpiresAt   time.Time
	DeviceLabel string
	Current     bool
}

type ListSessionsResult struct {
	Items []SessionSummary
	Page  int
	Size  int
	Total int
}

type RevokeSessionCommand struct {
	Identity    TrustedIdentity
	SessionID   string
	CurrentOnly bool
	CSRFToken   string
}

type RevokeSessionResult struct {
	SessionID  string
	Current    bool
	Processing *AcceptedSecurityOperation
}

type GetSecurityOperationQuery struct {
	Identity    TrustedIdentity
	OperationID string
}

type SecurityOperationResult struct {
	OperationID       string
	Type              string
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CompletedAt       *time.Time
	RetryAfterSeconds *int
	ErrorCode         *string
}

type AcceptedSecurityOperation struct {
	OperationID       string
	RetryAfterSeconds int
}

type Principal struct {
	AccountID        string
	UserID           string
	Email            string
	Roles            []string
	AccountStatus    string
	SessionID        string
	SessionVersion   int64
	PrincipalVersion int64
}

type TrustedIdentity struct {
	AccountID        string
	UserID           string
	SessionID        string
	SessionVersion   int64
	PrincipalVersion int64
	Roles            []string
}

type Handler struct {
	service Service
	router  *gin.Engine
}

func NewHandler(service Service) *gin.Engine {
	h := &Handler{
		service: service,
		router:  gin.New(),
	}
	h.routes()
	return h.router
}

func (h *Handler) routes() {
	h.router.POST("/api/v1/auth/register", h.register)
	h.router.POST("/api/v1/auth/login", h.login)
	h.router.POST("/api/v1/auth/refresh", h.refresh)
	h.router.POST("/api/v1/auth/logout", h.logout)
	h.router.GET("/api/v1/auth/me", h.me)
	h.router.GET("/api/v1/auth/csrf", h.csrf)
	h.router.GET("/api/v1/auth/sessions", h.listSessions)
	h.router.DELETE("/api/v1/auth/sessions/current", h.revokeCurrentSession)
	h.router.DELETE("/api/v1/auth/sessions/:sessionId", h.revokeSession)
	h.router.GET("/api/v1/auth/security-operations/:operationId", h.getSecurityOperation)
}

func (h *Handler) register(c *gin.Context) {
	w, r := c.Writer, c.Request
	var req struct {
		Email                  string `json:"email"`
		Nickname               string `json:"nickname"`
		Password               string `json:"password"`
		EmailVerificationToken string `json:"emailVerificationToken"`
	}
	if !decodeJSONBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Nickname) == "" || strings.TrimSpace(req.Password) == "" || strings.TrimSpace(req.EmailVerificationToken) == "" {
		writeValidationError(w)
		return
	}

	result, err := h.service.Register(r.Context(), RegisterCommand{
		Email:                  req.Email,
		Nickname:               req.Nickname,
		Password:               req.Password,
		EmailVerificationToken: req.EmailVerificationToken,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	if result.Authenticated {
		writeRefreshCookie(w, result.RefreshToken, result.RefreshTokenExpiresAt)
		writeCSRFCookie(w, result.CSRFToken)
	}
	writeJSON(w, http.StatusOK, sharedhttp.CodeSuccess, sharedhttp.MessageSuccess, map[string]any{
		"registered":          result.Registered,
		"authenticated":       result.Authenticated,
		"accessToken":         stringOrNil(result.AccessToken),
		"tokenType":           bearerOrNil(result.Authenticated),
		"expiresIn":           expiresInOrNil(result.Authenticated),
		"csrfToken":           stringOrNil(result.CSRFToken),
		"principal":           principalPayload(result.Principal, false),
		"loginDeferredReason": stringOrNil(result.LoginDeferredReason),
	})
}

func (h *Handler) login(c *gin.Context) {
	w, r := c.Writer, c.Request
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSONBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		writeValidationError(w)
		return
	}

	result, err := h.service.Login(r.Context(), LoginCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	writeRefreshCookie(w, result.RefreshToken, result.RefreshTokenExpiresAt)
	writeCSRFCookie(w, result.CSRFToken)
	sharedhttp.WriteSuccess(w, map[string]any{
		"accessToken": result.AccessToken,
		"tokenType":   "Bearer",
		"expiresIn":   fixedExpiresIn,
		"csrfToken":   result.CSRFToken,
		"principal":   principalPayload(result.Principal, false),
	})
}

func (h *Handler) refresh(c *gin.Context) {
	w, r := c.Writer, c.Request
	refreshToken, hasRefresh := cookieValue(r, refreshTokenCookieName)
	if !hasRefresh {
		writeMappedError(w, ErrTokenInvalid)
		return
	}
	csrfToken, ok := validateCSRFCookieRequest(r)
	if !ok {
		writeMappedError(w, ErrCSRFInvalid)
		return
	}

	result, err := h.service.Refresh(r.Context(), RefreshCommand{
		RefreshToken: refreshToken,
		CSRFToken:    csrfToken,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	if result.Processing != nil {
		writeAccepted(w, map[string]any{
			"operationId":       result.Processing.OperationID,
			"status":            "PROCESSING",
			"retryAfterSeconds": result.Processing.RetryAfterSeconds,
			"refreshAccepted":   false,
		})
		return
	}

	writeRefreshCookie(w, result.RefreshToken, result.RefreshTokenExpiresAt)
	writeCSRFCookie(w, result.CSRFToken)
	sharedhttp.WriteSuccess(w, map[string]any{
		"accessToken": result.AccessToken,
		"tokenType":   "Bearer",
		"expiresIn":   fixedExpiresIn,
		"csrfToken":   result.CSRFToken,
		"principal":   principalPayload(result.Principal, false),
	})
}

func (h *Handler) logout(c *gin.Context) {
	w, r := c.Writer, c.Request
	refreshToken, hasRefresh := cookieValue(r, refreshTokenCookieName)
	identity, hasIdentity := trustedIdentityFromRequest(r)
	csrfToken := ""

	// Requests without any trusted identity or refresh token cannot revoke a
	// server-side session, so logout stays a local cookie cleanup operation.
	if !hasIdentity && !hasRefresh {
		clearSessionCookies(w)
		sharedhttp.WriteSuccess(w, map[string]any{
			"loggedOut":     true,
			"serverRevoked": false,
		})
		return
	}

	// Any request that can trigger server-side revoke must pass double-submit CSRF.
	var ok bool
	csrfToken, ok = validateCSRFCookieRequest(r)
	if !ok {
		writeMappedError(w, ErrCSRFInvalid)
		return
	}

	result, err := h.service.Logout(r.Context(), LogoutCommand{
		Identity:     identity,
		HasIdentity:  hasIdentity,
		RefreshToken: refreshToken,
		CSRFToken:    csrfToken,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}

	clearSessionCookies(w)
	if result.Processing != nil {
		writeAccepted(w, map[string]any{
			"operationId":       result.Processing.OperationID,
			"status":            "PROCESSING",
			"retryAfterSeconds": result.Processing.RetryAfterSeconds,
			"loggedOut":         true,
		})
		return
	}

	sharedhttp.WriteSuccess(w, map[string]any{
		"loggedOut":     result.LoggedOut,
		"serverRevoked": result.ServerRevoked,
	})
}

func (h *Handler) me(c *gin.Context) {
	w, r := c.Writer, c.Request
	identity, ok := trustedIdentityFromRequest(r)
	if !ok {
		writeMappedError(w, ErrLoginRequired)
		return
	}

	principal, err := h.service.GetCurrentPrincipal(r.Context(), CurrentPrincipalQuery{Identity: identity})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, map[string]any{
		"principal": principalPayload(principal, true),
	})
}

func (h *Handler) csrf(c *gin.Context) {
	w, r := c.Writer, c.Request
	result, err := h.service.IssueCSRFToken(r.Context())
	if err != nil {
		writeMappedError(w, err)
		return
	}
	writeCSRFCookie(w, result.CSRFToken)
	sharedhttp.WriteSuccess(w, map[string]any{
		"csrfToken": result.CSRFToken,
	})
}

func (h *Handler) listSessions(c *gin.Context) {
	w, r := c.Writer, c.Request
	identity, ok := trustedIdentityFromRequest(r)
	if !ok {
		writeMappedError(w, ErrLoginRequired)
		return
	}

	page, size, valid := parsePagination(r)
	if !valid {
		writeValidationError(w)
		return
	}

	result, err := h.service.ListSessions(r.Context(), ListSessionsQuery{
		Identity: identity,
		Page:     page,
		Size:     size,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, map[string]any{
			"sessionId":   item.SessionID,
			"createdAt":   item.CreatedAt.Format(time.RFC3339),
			"lastSeenAt":  formatTimePtr(item.LastSeenAt),
			"expiresAt":   item.ExpiresAt.Format(time.RFC3339),
			"deviceLabel": stringOrNil(item.DeviceLabel),
			"current":     item.Current,
		})
	}
	sharedhttp.WriteSuccess(w, map[string]any{
		"items": items,
		"page":  result.Page,
		"size":  result.Size,
		"total": result.Total,
	})
}

func (h *Handler) revokeCurrentSession(c *gin.Context) {
	w, r := c.Writer, c.Request
	identity, ok := trustedIdentityFromRequest(r)
	if !ok {
		writeMappedError(w, ErrLoginRequired)
		return
	}
	csrfToken, ok := validateCSRFCookieRequest(r)
	if !ok {
		writeMappedError(w, ErrCSRFInvalid)
		return
	}

	result, err := h.service.RevokeSession(r.Context(), RevokeSessionCommand{
		Identity:    identity,
		SessionID:   identity.SessionID,
		CurrentOnly: true,
		CSRFToken:   csrfToken,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	h.writeRevokeSessionResponse(w, result)
}

func (h *Handler) revokeSession(c *gin.Context) {
	w, r := c.Writer, c.Request
	identity, ok := trustedIdentityFromRequest(r)
	if !ok {
		writeMappedError(w, ErrLoginRequired)
		return
	}
	csrfToken, ok := validateCSRFCookieRequest(r)
	if !ok {
		writeMappedError(w, ErrCSRFInvalid)
		return
	}
	sessionID := strings.TrimSpace(c.Param("sessionId"))
	if sessionID == "" {
		writeValidationError(w)
		return
	}

	result, err := h.service.RevokeSession(r.Context(), RevokeSessionCommand{
		Identity:    identity,
		SessionID:   sessionID,
		CurrentOnly: false,
		CSRFToken:   csrfToken,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	h.writeRevokeSessionResponse(w, result)
}

func (h *Handler) writeRevokeSessionResponse(w http.ResponseWriter, result RevokeSessionResult) {
	if result.Current {
		clearSessionCookies(w)
	}
	if result.Processing != nil {
		writeAccepted(w, map[string]any{
			"operationId":       result.Processing.OperationID,
			"status":            "PROCESSING",
			"retryAfterSeconds": result.Processing.RetryAfterSeconds,
			"sessionId":         result.SessionID,
		})
		return
	}
	sharedhttp.WriteSuccess(w, map[string]any{
		"status":    "REVOKED",
		"sessionId": result.SessionID,
		"current":   result.Current,
	})
}

func (h *Handler) getSecurityOperation(c *gin.Context) {
	w, r := c.Writer, c.Request
	identity, ok := trustedIdentityFromRequest(r)
	if !ok {
		writeMappedError(w, ErrLoginRequired)
		return
	}
	operationID := strings.TrimSpace(c.Param("operationId"))
	if operationID == "" {
		writeValidationError(w)
		return
	}

	result, err := h.service.GetSecurityOperation(r.Context(), GetSecurityOperationQuery{
		Identity:    identity,
		OperationID: operationID,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, map[string]any{
		"operationId":       result.OperationID,
		"type":              result.Type,
		"status":            result.Status,
		"createdAt":         result.CreatedAt.Format(time.RFC3339),
		"updatedAt":         result.UpdatedAt.Format(time.RFC3339),
		"completedAt":       formatTimePtr(result.CompletedAt),
		"retryAfterSeconds": intPtrValue(result.RetryAfterSeconds),
		"errorCode":         stringPtrValue(result.ErrorCode),
	})
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeValidationError(w)
		return false
	}
	return true
}

func parsePagination(r *http.Request) (int, int, bool) {
	page := 1
	size := defaultPageSize

	if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return 0, 0, false
		}
		page = value
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("size")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxPageSize {
			return 0, 0, false
		}
		size = value
	}
	return page, size, true
}

func trustedIdentityFromRequest(r *http.Request) (TrustedIdentity, bool) {
	accountID := strings.TrimSpace(r.Header.Get(accountIDHeaderName))
	userID := strings.TrimSpace(r.Header.Get(userIDHeaderName))
	sessionID := strings.TrimSpace(r.Header.Get(sessionIDHeaderName))
	if accountID == "" || userID == "" || sessionID == "" {
		return TrustedIdentity{}, false
	}

	sessionVersion, err := strconv.ParseInt(strings.TrimSpace(r.Header.Get(sessionVersionHeaderName)), 10, 64)
	if err != nil {
		return TrustedIdentity{}, false
	}
	principalVersion, err := strconv.ParseInt(strings.TrimSpace(r.Header.Get(principalVersionHeaderName)), 10, 64)
	if err != nil {
		return TrustedIdentity{}, false
	}

	return TrustedIdentity{
		AccountID:        accountID,
		UserID:           userID,
		SessionID:        sessionID,
		SessionVersion:   sessionVersion,
		PrincipalVersion: principalVersion,
		Roles:            splitRoles(r.Header.Get(userRolesHeaderName)),
	}, true
}

func splitRoles(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	roles := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		roles = append(roles, trimmed)
	}
	return roles
}

func validateCSRFCookieRequest(r *http.Request) (string, bool) {
	headerValue := strings.TrimSpace(r.Header.Get(csrfHeaderName))
	cookieValue, ok := cookieValue(r, csrfTokenCookieName)
	if !ok || headerValue == "" || headerValue != cookieValue {
		return "", false
	}
	return headerValue, true
}

func cookieValue(r *http.Request, name string) (string, bool) {
	cookie, err := r.Cookie(name)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return "", false
	}
	return cookie.Value, true
}

func principalPayload(principal Principal, includeSessionID bool) any {
	if principal.AccountID == "" {
		return nil
	}

	payload := map[string]any{
		"accountId":        principal.AccountID,
		"userId":           principal.UserID,
		"email":            principal.Email,
		"roles":            principal.RolesOrEmpty(),
		"accountStatus":    principal.AccountStatus,
		"sessionVersion":   principal.SessionVersion,
		"principalVersion": principal.PrincipalVersion,
	}
	if includeSessionID {
		payload["sessionId"] = principal.SessionID
	}
	return payload
}

func (p Principal) RolesOrEmpty() []string {
	if p.Roles == nil {
		return []string{}
	}
	return p.Roles
}

func stringOrNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func stringPtrValue(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return *value
}

func intPtrValue(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func formatTimePtr(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.Format(time.RFC3339)
}

func bearerOrNil(authenticated bool) any {
	if !authenticated {
		return nil
	}
	return "Bearer"
}

func expiresInOrNil(authenticated bool) any {
	if !authenticated {
		return nil
	}
	return fixedExpiresIn
}

func writeRefreshCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	if strings.TrimSpace(token) == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     authCookiePath,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func writeCSRFCookie(w http.ResponseWriter, token string) {
	if strings.TrimSpace(token) == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfTokenCookieName,
		Value:    token,
		Path:     authCookiePath,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     authCookiePath,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     csrfTokenCookieName,
		Value:    "",
		Path:     authCookiePath,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func writeValidationError(w http.ResponseWriter) {
	sharedhttp.WriteErrorCode(w, http.StatusBadRequest, 1001, "参数校验失败")
}

func writeAccepted(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusAccepted, sharedhttp.CodeSuccess, sharedhttp.MessageSuccess, data)
}

func writeMappedError(w http.ResponseWriter, err error) {
	status, code, message := errorMapping(err)
	sharedhttp.WriteErrorCode(w, status, code, message)
}

func errorMapping(err error) (int, int, string) {
	switch {
	case errors.Is(err, ErrEmailInvalid):
		return http.StatusBadRequest, 2010, "邮箱格式错误"
	case errors.Is(err, ErrPasswordInvalid):
		return http.StatusBadRequest, 2011, "密码不符合要求"
	case errors.Is(err, ErrEmailExists):
		return http.StatusConflict, 2009, "邮箱已被注册"
	case errors.Is(err, ErrRegisterPendingRetryable):
		return http.StatusServiceUnavailable, 2012, "注册暂时未完成，请稍后重试"
	case errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized, 2003, "登录失败"
	case errors.Is(err, ErrAccountDisabled):
		return http.StatusForbidden, 2004, "账号已禁用"
	case errors.Is(err, ErrAccountBanned):
		return http.StatusForbidden, 2019, "账号已被封禁"
	case errors.Is(err, ErrAccountLocked):
		return http.StatusForbidden, 2014, "账号已临时锁定"
	case errors.Is(err, ErrCSRFInvalid):
		return http.StatusForbidden, 2013, "CSRF 校验失败"
	case errors.Is(err, ErrTokenInvalid):
		return http.StatusUnauthorized, 2001, "Token无效"
	case errors.Is(err, ErrTokenExpired):
		return http.StatusUnauthorized, 2002, "Token已过期"
	case errors.Is(err, ErrTokenReplayed):
		return http.StatusUnauthorized, 2017, "登录凭证已被重复使用"
	case errors.Is(err, ErrSessionRevoked):
		return http.StatusUnauthorized, 2018, "会话已失效"
	case errors.Is(err, ErrLoginRequired):
		return http.StatusUnauthorized, 2006, "请先登录"
	case errors.Is(err, ErrPermissionDenied):
		return http.StatusForbidden, 2005, "权限不足"
	case errors.Is(err, ErrRoleRequired):
		return http.StatusForbidden, 2007, "需要特定角色"
	case errors.Is(err, ErrResourceAccessDenied):
		return http.StatusForbidden, 2008, "无权访问该资源"
	case errors.Is(err, ErrDataNotFound):
		return http.StatusNotFound, 1005, "数据不存在"
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests, 2015, "请求过于频繁"
	// Principal unavailable is an Auth-specific contract and must not collapse
	// into the generic degraded code used by other dependency failures.
	case errors.Is(err, ErrPrincipalUnavailable):
		return http.StatusServiceUnavailable, 2016, "登录状态暂时无法确认"
	case errors.Is(err, ErrServiceDegraded):
		return http.StatusServiceUnavailable, 1004, "服务暂时不可用"
	default:
		return http.StatusInternalServerError, 1000, "服务器内部错误"
	}
}

func writeJSON(w http.ResponseWriter, status int, code int, message string, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(sharedhttp.Response{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	})
}
