package httpapi

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
	"github.com/gin-gonic/gin"
)

func requireInternalCaller(c *gin.Context, operation string) error {
	// Internal endpoints 只接受带调用方身份和目标操作的服务间请求；
	// 缺失时按依赖不可用处理，避免退化成 public anonymous 行为。
	if strings.TrimSpace(c.GetHeader("X-Caller-Service")) == "" || c.GetHeader("X-Caller-Operation") != operation {
		return application.ErrDependencyUnavailable
	}
	return nil
}

func publicIDFromPath(c *gin.Context) (application.PublicID, error) {
	publicID := strings.TrimSpace(c.Param("publicId"))
	if !isValidPublicID(publicID) {
		return "", application.ErrPublicIDInvalid
	}
	return application.PublicID(publicID), nil
}

func trustedUserIDFromContext(c *gin.Context) (application.UserID, error) {
	raw := strings.TrimSpace(c.GetHeader(userIDHeaderName))
	if raw == "" {
		return 0, errLoginRequired
	}

	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || userID <= 0 {
		return 0, errLoginRequired
	}
	return application.UserID(userID), nil
}

func isValidPublicID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func decodeRelationshipPageQuery(c *gin.Context) (string, int, error) {
	cursor := strings.TrimSpace(c.Query("cursor"))
	limit, err := sharedhttp.ParsePositiveInt(c.Query("limit"), 0, 0)
	if err != nil {
		return "", 0, err
	}
	return cursor, limit, nil
}

func decodeJSONBody(c *gin.Context, out any) error {
	return sharedhttp.DecodeJSONBody(c.Request, out)
}

func decodeUpdateProfileBody(c *gin.Context, cmd *application.UpdateProfileCommand) error {
	var body map[string]json.RawMessage
	if err := sharedhttp.DecodeJSONBody(c.Request, &body); err != nil {
		return err
	}

	if raw, ok := body["nickname"]; ok {
		value, ok := decodeRequiredString(raw)
		if !ok {
			return errInvalidRequest
		}
		cmd.Nickname = &value
	}
	if raw, ok := body["avatarFileId"]; ok {
		value, ok := decodeAvatarFileID(raw)
		if !ok {
			return errInvalidRequest
		}
		cmd.AvatarFileID = &value
	}
	if raw, ok := body["bio"]; ok {
		value, ok := decodeRequiredString(raw)
		if !ok {
			return errInvalidRequest
		}
		cmd.Bio = &value
	}
	if raw, ok := body["strangerMessageAllowed"]; ok {
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		cmd.StrangerMessageAllowed = &value
	}
	return nil
}

func decodeRequiredString(raw json.RawMessage) (string, bool) {
	if strings.TrimSpace(string(raw)) == "null" {
		return "", false
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func decodeAvatarFileID(raw json.RawMessage) (string, bool) {
	if strings.TrimSpace(string(raw)) == "null" {
		return "", true
	}
	return decodeRequiredString(raw)
}
