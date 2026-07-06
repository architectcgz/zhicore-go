package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
)

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errInvalidRequest
	}
	return nil
}

func parsePagination(r *http.Request) (int, int, bool) {
	page, err := sharedhttp.ParsePositiveInt(r.URL.Query().Get("page"), 1, 0)
	if err != nil {
		return 0, 0, false
	}
	size, err := sharedhttp.ParsePositiveInt(r.URL.Query().Get("size"), defaultPageSize, maxPageSize)
	if err != nil {
		return 0, 0, false
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
