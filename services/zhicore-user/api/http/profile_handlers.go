package httpapi

import (
	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) getMe(c *gin.Context) {
	userID, err := trustedUserIDFromContext(c)
	if err != nil {
		writeMappedError(c, err)
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
	publicID, err := publicIDFromPath(c)
	if err != nil {
		writeValidationError(c)
		return
	}

	profile, err := h.service.GetUserProfileByPublicID(c.Request.Context(), publicID)
	if err != nil {
		writeMappedError(c, err)
		return
	}

	sharedhttp.WriteSuccess(c.Writer, h.profileResponse(c.Request.Context(), profile))
}

func (h *Handler) updateProfile(c *gin.Context) {
	userID, err := trustedUserIDFromContext(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}

	// PATCH 允许省略资料字段，但当前操作者只能来自可信 header；
	// body 里的 userId/actor 类字段即使存在也必须被忽略，不能覆盖身份。
	cmd := application.UpdateProfileCommand{
		UserID: userID,
	}
	if err := decodeUpdateProfileBody(c, &cmd); err != nil {
		writeValidationError(c)
		return
	}

	updated, err := h.service.UpdateProfile(c.Request.Context(), cmd)
	if err != nil {
		writeMappedError(c, err)
		return
	}

	sharedhttp.WriteSuccess(c.Writer, h.profileResponse(c.Request.Context(), updated))
}
