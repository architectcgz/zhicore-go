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

	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
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
	BatchGetUserSimple(ctx context.Context, userIDs []application.UserID) (application.BatchUserSimpleResult, error)
	BatchGetUserAvailability(ctx context.Context, userIDs []application.UserID) ([]application.UserAvailability, error)
	BatchCheckBlocked(ctx context.Context, pairs []application.UserPair) (map[application.UserPair]bool, error)
}

type AvatarURLResolver interface {
	ResolveAvatarURL(ctx context.Context, fileID string) (string, error)
}

type Handler struct {
	service  Service
	resolver AvatarURLResolver
	router   *gin.Engine
}

func NewHandler(service Service, resolver AvatarURLResolver) *gin.Engine {
	h := &Handler{
		service:  service,
		resolver: resolver,
		router:   gin.New(),
	}
	h.routes()
	return h.router
}

func (h *Handler) routes() {
	h.router.GET("/api/v1/users/me", h.getMe)
	h.router.GET("/api/v1/users/me/blocked", h.listBlockedUsers)
	h.router.GET("/api/v1/users/:publicId", h.getProfile)
	h.router.PATCH("/api/v1/users/me/profile", h.updateProfile)
	h.router.POST("/api/v1/users/:publicId/block", h.blockUser)
	h.router.DELETE("/api/v1/users/:publicId/block", h.unblockUser)
	h.router.POST("/api/v1/users/:publicId/follow", h.followUser)
	h.router.DELETE("/api/v1/users/:publicId/follow", h.unfollowUser)
	h.router.GET("/api/v1/users/:publicId/followers", h.listFollowers)
	h.router.GET("/api/v1/users/:publicId/following", h.listFollowing)
	h.router.POST(usercontract.BatchAvailabilityPath, h.batchAvailability)
	h.router.POST(usercontract.BatchSimplePath, h.batchSimple)
	h.router.POST(usercontract.BatchCheckBlockedPath, h.batchCheckBlocked)
}

func (h *Handler) getMe(c *gin.Context) {
	userID, ok := trustedUserIDFromContext(c)
	if !ok {
		writeMappedError(c, errLoginRequired)
		return
	}

	profile, err := h.service.GetMyProfile(c.Request.Context(), userID)
	if err != nil {
		writeMappedError(c, err)
		return
	}

	sharedhttp.WriteSuccess(c.Writer, h.profileResponse(c.Request.Context(), profile))
}

func (h *Handler) getProfile(c *gin.Context) {
	publicID := strings.TrimSpace(c.Param("publicId"))
	if !isValidPublicID(publicID) {
		writeValidationError(c)
		return
	}

	profile, err := h.service.GetUserProfileByPublicID(c.Request.Context(), application.PublicID(publicID))
	if err != nil {
		writeMappedError(c, err)
		return
	}

	sharedhttp.WriteSuccess(c.Writer, h.profileResponse(c.Request.Context(), profile))
}

func (h *Handler) updateProfile(c *gin.Context) {
	userID, ok := trustedUserIDFromContext(c)
	if !ok {
		writeMappedError(c, errLoginRequired)
		return
	}

	// PATCH 允许省略资料字段，但当前操作者只能来自可信 header；
	// body 里的 userId/actor 类字段即使存在也必须被忽略，不能覆盖身份。
	cmd := application.UpdateProfileCommand{
		UserID: userID,
	}
	if !decodeUpdateProfileBody(c, &cmd) {
		return
	}

	updated, err := h.service.UpdateProfile(c.Request.Context(), cmd)
	if err != nil {
		writeMappedError(c, err)
		return
	}

	sharedhttp.WriteSuccess(c.Writer, h.profileResponse(c.Request.Context(), updated))
}

func (h *Handler) blockUser(c *gin.Context) {
	userID, ok := trustedUserIDFromContext(c)
	if !ok {
		writeMappedError(c, errLoginRequired)
		return
	}
	publicID, ok := publicIDFromPath(c)
	if !ok {
		return
	}
	if err := h.service.BlockUser(c.Request.Context(), application.BlockUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, map[string]bool{"blocked": true})
}

func (h *Handler) unblockUser(c *gin.Context) {
	userID, ok := trustedUserIDFromContext(c)
	if !ok {
		writeMappedError(c, errLoginRequired)
		return
	}
	publicID, ok := publicIDFromPath(c)
	if !ok {
		return
	}
	if err := h.service.UnblockUser(c.Request.Context(), application.UnblockUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, map[string]bool{"blocked": false})
}

func (h *Handler) followUser(c *gin.Context) {
	userID, ok := trustedUserIDFromContext(c)
	if !ok {
		writeMappedError(c, errLoginRequired)
		return
	}
	publicID, ok := publicIDFromPath(c)
	if !ok {
		return
	}
	if err := h.service.FollowUser(c.Request.Context(), application.FollowUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, map[string]bool{"following": true})
}

func (h *Handler) unfollowUser(c *gin.Context) {
	userID, ok := trustedUserIDFromContext(c)
	if !ok {
		writeMappedError(c, errLoginRequired)
		return
	}
	publicID, ok := publicIDFromPath(c)
	if !ok {
		return
	}
	if err := h.service.UnfollowUser(c.Request.Context(), application.UnfollowUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, map[string]bool{"following": false})
}

func (h *Handler) listBlockedUsers(c *gin.Context) {
	userID, ok := trustedUserIDFromContext(c)
	if !ok {
		writeMappedError(c, errLoginRequired)
		return
	}
	cursor, limit, ok := decodeRelationshipPageQuery(c)
	if !ok {
		return
	}
	page, err := h.service.ListBlockedUsers(c.Request.Context(), application.ListBlockedUsersQuery{ActorUserID: userID, Cursor: cursor, Limit: limit})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, h.relationshipPageResponse(c.Request.Context(), page))
}

func (h *Handler) listFollowers(c *gin.Context) {
	publicID, ok := publicIDFromPath(c)
	if !ok {
		return
	}
	cursor, limit, ok := decodeRelationshipPageQuery(c)
	if !ok {
		return
	}
	page, err := h.service.ListFollowers(c.Request.Context(), application.ListFollowersQuery{TargetPublicID: publicID, Cursor: cursor, Limit: limit})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, h.relationshipPageResponse(c.Request.Context(), page))
}

func (h *Handler) listFollowing(c *gin.Context) {
	publicID, ok := publicIDFromPath(c)
	if !ok {
		return
	}
	cursor, limit, ok := decodeRelationshipPageQuery(c)
	if !ok {
		return
	}
	page, err := h.service.ListFollowing(c.Request.Context(), application.ListFollowingQuery{TargetPublicID: publicID, Cursor: cursor, Limit: limit})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, h.relationshipPageResponse(c.Request.Context(), page))
}

func (h *Handler) batchAvailability(c *gin.Context) {
	if !requireInternalCaller(c, usercontract.OperationCommentCheckUserAvailability) {
		return
	}
	var req usercontract.IDsRequest
	if !decodeJSONBody(c, &req) {
		return
	}
	items, err := h.service.BatchGetUserAvailability(c.Request.Context(), applicationUserIDs(req.UserIDs))
	if err != nil {
		writeMappedError(c, err)
		return
	}
	resp := usercontract.AvailabilityBatchResponse{Items: make([]usercontract.AvailabilityItem, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, usercontract.AvailabilityItem{
			UserID:    int64(item.UserID),
			Available: item.Available,
			Status:    string(item.Status),
		})
	}
	sharedhttp.WriteSuccess(c.Writer, resp)
}

func (h *Handler) batchSimple(c *gin.Context) {
	if !requireInternalCaller(c, usercontract.OperationCommentBatchGetAuthorSummaries) {
		return
	}
	var req usercontract.IDsRequest
	if !decodeJSONBody(c, &req) {
		return
	}
	result, err := h.service.BatchGetUserSimple(c.Request.Context(), applicationUserIDs(req.UserIDs))
	if err != nil {
		writeMappedError(c, err)
		return
	}
	resp := usercontract.SimpleBatchResponse{
		Items:          make([]usercontract.SimpleUser, 0, len(result.Items)),
		MissingUserIDs: make([]int64, 0, len(result.MissingUserIDs)),
	}
	for _, item := range result.Items {
		resp.Items = append(resp.Items, h.simpleUserResponse(c.Request.Context(), item))
	}
	for _, userID := range result.MissingUserIDs {
		resp.MissingUserIDs = append(resp.MissingUserIDs, int64(userID))
	}
	sharedhttp.WriteSuccess(c.Writer, resp)
}

func (h *Handler) batchCheckBlocked(c *gin.Context) {
	if !requireInternalCaller(c, usercontract.OperationCommentBatchCheckBlocked) {
		return
	}
	var req usercontract.BlockPairsRequest
	if !decodeJSONBody(c, &req) {
		return
	}
	pairs := make([]application.UserPair, 0, len(req.Pairs))
	for _, pair := range req.Pairs {
		pairs = append(pairs, application.UserPair{
			ActorID:  application.UserID(pair.BlockerID),
			TargetID: application.UserID(pair.BlockedID),
		})
	}
	checked, err := h.service.BatchCheckBlocked(c.Request.Context(), pairs)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	resp := usercontract.BlockPairsResponse{Items: make([]usercontract.BlockPairResult, 0, len(pairs))}
	for _, pair := range pairs {
		resp.Items = append(resp.Items, usercontract.BlockPairResult{
			BlockerID: int64(pair.ActorID),
			BlockedID: int64(pair.TargetID),
			Blocked:   checked[pair],
		})
	}
	sharedhttp.WriteSuccess(c.Writer, resp)
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

func (h *Handler) simpleUserResponse(ctx context.Context, item application.UserSimple) usercontract.SimpleUser {
	resp := usercontract.SimpleUser{
		UserID:         int64(item.UserID),
		PublicID:       string(item.PublicID),
		Nickname:       item.Nickname,
		AvatarFileID:   item.AvatarFileID,
		ProfileVersion: item.ProfileVersion,
		Status:         string(item.Status),
	}
	if h.resolver == nil || strings.TrimSpace(item.AvatarFileID) == "" {
		return resp
	}
	url, err := h.resolver.ResolveAvatarURL(ctx, item.AvatarFileID)
	if err == nil && strings.TrimSpace(url) != "" {
		resp.AvatarURL = url
	}
	return resp
}

func applicationUserIDs(ids []int64) []application.UserID {
	result := make([]application.UserID, 0, len(ids))
	for _, id := range ids {
		result = append(result, application.UserID(id))
	}
	return result
}

func requireInternalCaller(c *gin.Context, operation string) bool {
	// Internal endpoints 只接受带调用方身份和目标操作的服务间请求；
	// 缺失时按依赖不可用处理，避免退化成 public anonymous 行为。
	if strings.TrimSpace(c.GetHeader("X-Caller-Service")) == "" || c.GetHeader("X-Caller-Operation") != operation {
		writeMappedError(c, application.ErrDependencyUnavailable)
		return false
	}
	return true
}

func publicIDFromPath(c *gin.Context) (application.PublicID, bool) {
	publicID := strings.TrimSpace(c.Param("publicId"))
	if !isValidPublicID(publicID) {
		writeValidationError(c)
		return "", false
	}
	return application.PublicID(publicID), true
}

func trustedUserIDFromContext(c *gin.Context) (application.UserID, bool) {
	raw := strings.TrimSpace(c.GetHeader(userIDHeaderName))
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

func decodeRelationshipPageQuery(c *gin.Context) (string, int, bool) {
	cursor := strings.TrimSpace(c.Query("cursor"))
	limit := 0
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeValidationError(c)
			return "", 0, false
		}
		limit = parsed
	}
	return cursor, limit, true
}

func decodeJSONBody(c *gin.Context, out any) bool {
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(out); err != nil {
		writeValidationError(c)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeValidationError(c)
		return false
	}
	return true
}

func decodeUpdateProfileBody(c *gin.Context, cmd *application.UpdateProfileCommand) bool {
	var body map[string]json.RawMessage
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(&body); err != nil {
		writeValidationError(c)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeValidationError(c)
		return false
	}

	if raw, ok := body["nickname"]; ok {
		value, ok := decodeRequiredString(raw)
		if !ok {
			writeValidationError(c)
			return false
		}
		cmd.Nickname = &value
	}
	if raw, ok := body["avatarFileId"]; ok {
		value, ok := decodeAvatarFileID(raw)
		if !ok {
			writeValidationError(c)
			return false
		}
		cmd.AvatarFileID = &value
	}
	if raw, ok := body["bio"]; ok {
		value, ok := decodeRequiredString(raw)
		if !ok {
			writeValidationError(c)
			return false
		}
		cmd.Bio = &value
	}
	if raw, ok := body["strangerMessageAllowed"]; ok {
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			writeValidationError(c)
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

func writeValidationError(c *gin.Context) {
	sharedhttp.WriteErrorCode(c.Writer, http.StatusBadRequest, 1001, "参数校验失败")
}

func writeMappedError(c *gin.Context, err error) {
	status, code, message := errorMapping(err)
	sharedhttp.WriteErrorCode(c.Writer, status, code, message)
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
