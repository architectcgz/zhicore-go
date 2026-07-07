package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

func actorFromRequest(r *http.Request) (*application.Actor, error) {
	raw := strings.TrimSpace(r.Header.Get(userIDHeaderName))
	if raw == "" {
		return nil, errLoginRequired
	}
	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || userID <= 0 {
		return nil, errLoginRequired
	}
	return &application.Actor{UserID: userID, Roles: rolesFromRequest(r)}, nil
}

func optionalActorFromRequest(r *http.Request) (*application.Actor, error) {
	if strings.TrimSpace(r.Header.Get(userIDHeaderName)) == "" {
		return nil, nil
	}
	return actorFromRequest(r)
}

func requireAdminActorFromRequest(r *http.Request) (*application.Actor, error) {
	actor, err := actorFromRequest(r)
	if err != nil {
		return nil, err
	}
	if !actor.HasRole("admin") && !actor.HasRole("role_admin") {
		return nil, application.ErrRoleRequired
	}
	return actor, nil
}

func rolesFromRequest(r *http.Request) []string {
	raw := strings.TrimSpace(r.Header.Get(userRolesHeaderName))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	roles := make([]string, 0, len(parts))
	for _, part := range parts {
		if role := strings.TrimSpace(part); role != "" {
			roles = append(roles, role)
		}
	}
	return roles
}

func postIDFromPath(c *gin.Context) (string, error) {
	postID := strings.TrimSpace(c.Param("postId"))
	if postID == "" {
		return "", application.ErrInvalidArgument
	}
	return postID, nil
}

func sessionIDFromPath(c *gin.Context) (string, error) {
	sessionID := strings.TrimSpace(c.Param("sessionId"))
	if sessionID == "" {
		return "", application.ErrInvalidArgument
	}
	return sessionID, nil
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) error {
	return sharedhttp.DecodeJSONBodyLimited(w, r, maxJSONRequestBodyBytes, target)
}

func optionalPositiveIntQuery(c *gin.Context, key string) (int, error) {
	value, err := sharedhttp.ParsePositiveInt(c.Query(key), 0, 0)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func writeDecodeError(w http.ResponseWriter, err error) {
	if isRequestBodyTooLarge(err) {
		sharedhttp.WriteErrorCode(w, http.StatusRequestEntityTooLarge, 4015, "Body too large")
		return
	}
	writeValidationError(w)
}

func isRequestBodyTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}
