package httpapi

import (
	"net/http"
	"strings"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/gin-gonic/gin"
)

func (h *Handler) register(c *gin.Context) {
	w, r := c.Writer, c.Request
	var req struct {
		Email                  string `json:"email"`
		Nickname               string `json:"nickname"`
		Password               string `json:"password"`
		EmailVerificationToken string `json:"emailVerificationToken"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeValidationError(w)
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
		Email      string `json:"email"`
		Password   string `json:"password"`
		RememberMe *bool  `json:"rememberMe"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeValidationError(w)
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" || req.RememberMe == nil {
		writeValidationError(w)
		return
	}

	result, err := h.service.Login(r.Context(), LoginCommand{
		Email:      req.Email,
		Password:   req.Password,
		RememberMe: *req.RememberMe,
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
