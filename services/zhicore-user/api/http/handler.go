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
)

const userIDHeaderName = "X-User-Id"

var errLoginRequired = errors.New("login required")

type Service interface {
	GetMyProfile(ctx context.Context, userID application.UserID) (application.Profile, error)
	GetUserProfileByPublicID(ctx context.Context, publicID application.PublicID) (application.Profile, error)
	UpdateProfile(ctx context.Context, cmd application.UpdateProfileCommand) (application.Profile, error)
}

type AvatarURLResolver interface {
	ResolveAvatarURL(ctx context.Context, fileID string) (string, error)
}

type Handler struct {
	service  Service
	resolver AvatarURLResolver
	mux      *http.ServeMux
}

func NewHandler(service Service, resolver AvatarURLResolver) http.Handler {
	h := &Handler{
		service:  service,
		resolver: resolver,
		mux:      http.NewServeMux(),
	}
	h.routes()
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) routes() {
	h.mux.HandleFunc("GET /api/v1/users/me", h.getMe)
	h.mux.HandleFunc("GET /api/v1/users/{publicId}", h.getProfile)
	h.mux.HandleFunc("PATCH /api/v1/users/me/profile", h.updateProfile)
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

type userProfileResp struct {
	PublicID               string `json:"publicId"`
	Nickname               string `json:"nickname"`
	AvatarFileID           string `json:"avatarFileId,omitempty"`
	AvatarURL              string `json:"avatarUrl,omitempty"`
	Bio                    string `json:"bio,omitempty"`
	StrangerMessageAllowed bool   `json:"strangerMessageAllowed"`
	ProfileVersion         int64  `json:"profileVersion"`
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
	case errors.Is(err, application.ErrDependencyUnavailable):
		return http.StatusServiceUnavailable, 1004, "服务暂时不可用"
	default:
		return http.StatusInternalServerError, 1000, "服务器内部错误"
	}
}
