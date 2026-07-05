package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
	"github.com/gin-gonic/gin"
)

const userIDHeaderName = "X-User-Id"

var errLoginRequired = errors.New("login required")

type Service interface {
	GetMyProfile(ctx context.Context, userID application.UserID) (application.Profile, error)
	GetUserProfileByPublicID(ctx context.Context, publicID application.PublicID) (application.Profile, error)
	UpdateProfile(ctx context.Context, cmd application.UpdateProfileCommand) (application.Profile, error)
	BlockUser(ctx context.Context, cmd application.BlockUserCommand) error
	UnblockUser(ctx context.Context, cmd application.UnblockUserCommand) error
	ListBlockedUsers(ctx context.Context, query application.ListBlockedUsersQuery) (application.RelationshipProfilePage, error)
	FollowUser(ctx context.Context, cmd application.FollowUserCommand) error
	UnfollowUser(ctx context.Context, cmd application.UnfollowUserCommand) error
	ListFollowers(ctx context.Context, query application.ListFollowersQuery) (application.RelationshipProfilePage, error)
	ListFollowing(ctx context.Context, query application.ListFollowingQuery) (application.RelationshipProfilePage, error)
}

type AvatarURLResolver interface {
	ResolveAvatarURL(ctx context.Context, fileID string) (string, error)
}

type Handler struct {
	service  Service
	resolver AvatarURLResolver
	router   *gin.Engine
}

func NewHandler(service Service, resolver AvatarURLResolver) http.Handler {
	h := &Handler{
		service:  service,
		resolver: resolver,
		router:   gin.New(),
	}
	h.routes()
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) routes() {
	h.router.GET("/api/v1/users/me", ginHTTPHandler(h.getMe))
	h.router.GET("/api/v1/users/me/blocked", ginHTTPHandler(h.listBlockedUsers))
	h.router.GET("/api/v1/users/:publicId", ginHTTPHandler(h.getProfile))
	h.router.PATCH("/api/v1/users/me/profile", ginHTTPHandler(h.updateProfile))
	h.router.POST("/api/v1/users/:publicId/block", ginHTTPHandler(h.blockUser))
	h.router.DELETE("/api/v1/users/:publicId/block", ginHTTPHandler(h.unblockUser))
	h.router.POST("/api/v1/users/:publicId/follow", ginHTTPHandler(h.followUser))
	h.router.DELETE("/api/v1/users/:publicId/follow", ginHTTPHandler(h.unfollowUser))
	h.router.GET("/api/v1/users/:publicId/followers", ginHTTPHandler(h.listFollowers))
	h.router.GET("/api/v1/users/:publicId/following", ginHTTPHandler(h.listFollowing))
}

func ginHTTPHandler(next http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Path params stay on net/http.Request so Gin does not leak past the
		// HTTP adapter and application inputs remain explicit DTOs.
		for _, param := range c.Params {
			c.Request.SetPathValue(param.Key, param.Value)
		}
		next(c.Writer, c.Request)
	}
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := trustedUserIDFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}

	profile, err := h.service.GetMyProfile(r.Context(), userID)
	if err != nil {
		writeMappedError(w, err)
		return
	}

	sharedhttp.WriteSuccess(w, h.profileResponse(r.Context(), profile))
}

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	publicID := strings.TrimSpace(r.PathValue("publicId"))
	if !isValidPublicID(publicID) {
		writeValidationError(w)
		return
	}

	profile, err := h.service.GetUserProfileByPublicID(r.Context(), application.PublicID(publicID))
	if err != nil {
		writeMappedError(w, err)
		return
	}

	sharedhttp.WriteSuccess(w, h.profileResponse(r.Context(), profile))
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := trustedUserIDFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}

	// PATCH 允许省略资料字段，但当前操作者只能来自可信 header；
	// body 里的 userId/actor 类字段即使存在也必须被忽略，不能覆盖身份。
	cmd := application.UpdateProfileCommand{
		UserID: userID,
	}
	if !decodeUpdateProfileBody(w, r, &cmd) {
		return
	}

	updated, err := h.service.UpdateProfile(r.Context(), cmd)
	if err != nil {
		writeMappedError(w, err)
		return
	}

	sharedhttp.WriteSuccess(w, h.profileResponse(r.Context(), updated))
}

func (h *Handler) blockUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := trustedUserIDFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	publicID, ok := publicIDFromPath(w, r)
	if !ok {
		return
	}
	if err := h.service.BlockUser(r.Context(), application.BlockUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, map[string]bool{"blocked": true})
}

func (h *Handler) unblockUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := trustedUserIDFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	publicID, ok := publicIDFromPath(w, r)
	if !ok {
		return
	}
	if err := h.service.UnblockUser(r.Context(), application.UnblockUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, map[string]bool{"blocked": false})
}

func (h *Handler) followUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := trustedUserIDFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	publicID, ok := publicIDFromPath(w, r)
	if !ok {
		return
	}
	if err := h.service.FollowUser(r.Context(), application.FollowUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, map[string]bool{"following": true})
}

func (h *Handler) unfollowUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := trustedUserIDFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	publicID, ok := publicIDFromPath(w, r)
	if !ok {
		return
	}
	if err := h.service.UnfollowUser(r.Context(), application.UnfollowUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, map[string]bool{"following": false})
}

func (h *Handler) listBlockedUsers(w http.ResponseWriter, r *http.Request) {
	userID, ok := trustedUserIDFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	cursor, limit, ok := decodeRelationshipPageQuery(w, r)
	if !ok {
		return
	}
	page, err := h.service.ListBlockedUsers(r.Context(), application.ListBlockedUsersQuery{ActorUserID: userID, Cursor: cursor, Limit: limit})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, h.relationshipPageResponse(r.Context(), page))
}

func (h *Handler) listFollowers(w http.ResponseWriter, r *http.Request) {
	publicID, ok := publicIDFromPath(w, r)
	if !ok {
		return
	}
	cursor, limit, ok := decodeRelationshipPageQuery(w, r)
	if !ok {
		return
	}
	page, err := h.service.ListFollowers(r.Context(), application.ListFollowersQuery{TargetPublicID: publicID, Cursor: cursor, Limit: limit})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, h.relationshipPageResponse(r.Context(), page))
}

func (h *Handler) listFollowing(w http.ResponseWriter, r *http.Request) {
	publicID, ok := publicIDFromPath(w, r)
	if !ok {
		return
	}
	cursor, limit, ok := decodeRelationshipPageQuery(w, r)
	if !ok {
		return
	}
	page, err := h.service.ListFollowing(r.Context(), application.ListFollowingQuery{TargetPublicID: publicID, Cursor: cursor, Limit: limit})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, h.relationshipPageResponse(r.Context(), page))
}

type userProfileResp struct {
	PublicID               string `json:"publicId"`
	Nickname               string `json:"nickname"`
	AvatarFileID           string `json:"avatarFileId,omitempty"`
	AvatarURL              string `json:"avatarUrl,omitempty"`
	Bio                    string `json:"bio,omitempty"`
	StrangerMessageAllowed bool   `json:"strangerMessageAllowed"`
	ProfileVersion         int64  `json:"profileVersion"`
}

type relationshipPageResp struct {
	Items      []userProfileResp `json:"items"`
	NextCursor string            `json:"nextCursor,omitempty"`
	HasMore    bool              `json:"hasMore"`
}

func (h *Handler) profileResponse(ctx context.Context, profile application.Profile) userProfileResp {
	resp := userProfileResp{
		PublicID:               string(profile.PublicID),
		Nickname:               profile.Nickname,
		AvatarFileID:           profile.AvatarFileID,
		Bio:                    profile.Bio,
		StrangerMessageAllowed: profile.StrangerMessageAllowed,
		ProfileVersion:         profile.ProfileVersion,
	}
	if h.resolver == nil || strings.TrimSpace(profile.AvatarFileID) == "" {
		return resp
	}

	// avatarUrl 是读取时派生的展示字段；File 解析失败时仍返回 profile 事实，
	// 只省略 avatarUrl，避免把非事实字段的故障升级成整个资料查询失败。
	url, err := h.resolver.ResolveAvatarURL(ctx, profile.AvatarFileID)
	if err == nil && strings.TrimSpace(url) != "" {
		resp.AvatarURL = url
	}
	return resp
}

func (h *Handler) relationshipPageResponse(ctx context.Context, page application.RelationshipProfilePage) relationshipPageResp {
	items := make([]userProfileResp, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, h.profileResponse(ctx, item))
	}
	return relationshipPageResp{
		Items:      items,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	}
}

func publicIDFromPath(w http.ResponseWriter, r *http.Request) (application.PublicID, bool) {
	publicID := strings.TrimSpace(r.PathValue("publicId"))
	if !isValidPublicID(publicID) {
		writeValidationError(w)
		return "", false
	}
	return application.PublicID(publicID), true
}

func trustedUserIDFromRequest(r *http.Request) (application.UserID, bool) {
	raw := strings.TrimSpace(r.Header.Get(userIDHeaderName))
	if raw == "" {
		return 0, false
	}

	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}
	return application.UserID(userID), true
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

func decodeRelationshipPageQuery(w http.ResponseWriter, r *http.Request) (string, int, bool) {
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	limit := 0
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeValidationError(w)
			return "", 0, false
		}
		limit = parsed
	}
	return cursor, limit, true
}

func decodeUpdateProfileBody(w http.ResponseWriter, r *http.Request, cmd *application.UpdateProfileCommand) bool {
	var body map[string]json.RawMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		writeValidationError(w)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeValidationError(w)
		return false
	}

	if raw, ok := body["nickname"]; ok {
		value, ok := decodeRequiredString(raw)
		if !ok {
			writeValidationError(w)
			return false
		}
		cmd.Nickname = &value
	}
	if raw, ok := body["avatarFileId"]; ok {
		value, ok := decodeAvatarFileID(raw)
		if !ok {
			writeValidationError(w)
			return false
		}
		cmd.AvatarFileID = &value
	}
	if raw, ok := body["bio"]; ok {
		value, ok := decodeRequiredString(raw)
		if !ok {
			writeValidationError(w)
			return false
		}
		cmd.Bio = &value
	}
	if raw, ok := body["strangerMessageAllowed"]; ok {
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			writeValidationError(w)
			return false
		}
		cmd.StrangerMessageAllowed = &value
	}
	return true
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

func writeValidationError(w http.ResponseWriter) {
	sharedhttp.WriteErrorCode(w, http.StatusBadRequest, 1001, "参数校验失败")
}

func writeMappedError(w http.ResponseWriter, err error) {
	status, code, message := errorMapping(err)
	sharedhttp.WriteErrorCode(w, status, code, message)
}

func errorMapping(err error) (int, int, string) {
	switch {
	case errors.Is(err, errLoginRequired):
		return http.StatusUnauthorized, 2006, "请先登录"
	case errors.Is(err, application.ErrPublicIDInvalid):
		return http.StatusBadRequest, 1001, "参数校验失败"
	case errors.Is(err, application.ErrProfileNotFound):
		return http.StatusNotFound, 3001, "用户不存在"
	case errors.Is(err, application.ErrNicknameTaken):
		return http.StatusConflict, 3005, "昵称已被使用"
	case errors.Is(err, application.ErrUserNotActive):
		return http.StatusForbidden, 3006, "用户不可用"
	case errors.Is(err, application.ErrNicknameInvalid):
		return http.StatusBadRequest, 3013, "昵称不合法"
	case errors.Is(err, application.ErrBioInvalid):
		return http.StatusBadRequest, 3014, "简介不合法"
	case errors.Is(err, application.ErrAvatarInvalid):
		return http.StatusBadRequest, 3015, "头像文件不可引用"
	case errors.Is(err, application.ErrCannotFollowSelf):
		return http.StatusBadRequest, 3007, "不能关注自己"
	case errors.Is(err, application.ErrInteractionBlocked):
		return http.StatusForbidden, 3010, "互动被拉黑阻止"
	case errors.Is(err, application.ErrCannotBlockSelf):
		return http.StatusBadRequest, 3011, "不能拉黑自己"
	case errors.Is(err, application.ErrCursorInvalid):
		return http.StatusBadRequest, 1001, "参数校验失败"
	case errors.Is(err, application.ErrDependencyUnavailable):
		return http.StatusServiceUnavailable, 1004, "服务暂时不可用"
	default:
		return http.StatusInternalServerError, 1000, "服务器内部错误"
	}
}
