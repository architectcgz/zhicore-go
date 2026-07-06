package httpapi

import (
	"net/http"
	"strings"
	"time"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/gin-gonic/gin"
)

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
