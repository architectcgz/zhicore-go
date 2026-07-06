package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
)

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
	if value == nil {
		return nil
	}
	formatted := sharedhttp.FormatRFC3339UTC(*value)
	if formatted == "" {
		return nil
	}
	return formatted
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
