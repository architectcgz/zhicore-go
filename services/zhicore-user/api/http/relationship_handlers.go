package httpapi

import (
	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) blockUser(c *gin.Context) {
	userID, err := trustedUserIDFromContext(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	publicID, err := publicIDFromPath(c)
	if err != nil {
		writeValidationError(c)
		return
	}
	if err := h.service.BlockUser(c.Request.Context(), application.BlockUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, map[string]bool{"blocked": true})
}

func (h *Handler) unblockUser(c *gin.Context) {
	userID, err := trustedUserIDFromContext(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	publicID, err := publicIDFromPath(c)
	if err != nil {
		writeValidationError(c)
		return
	}
	if err := h.service.UnblockUser(c.Request.Context(), application.UnblockUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, map[string]bool{"blocked": false})
}

func (h *Handler) followUser(c *gin.Context) {
	userID, err := trustedUserIDFromContext(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	publicID, err := publicIDFromPath(c)
	if err != nil {
		writeValidationError(c)
		return
	}
	if err := h.service.FollowUser(c.Request.Context(), application.FollowUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, map[string]bool{"following": true})
}

func (h *Handler) unfollowUser(c *gin.Context) {
	userID, err := trustedUserIDFromContext(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	publicID, err := publicIDFromPath(c)
	if err != nil {
		writeValidationError(c)
		return
	}
	if err := h.service.UnfollowUser(c.Request.Context(), application.UnfollowUserCommand{ActorUserID: userID, TargetPublicID: publicID}); err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, map[string]bool{"following": false})
}

func (h *Handler) listBlockedUsers(c *gin.Context) {
	userID, err := trustedUserIDFromContext(c)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	cursor, limit, err := decodeRelationshipPageQuery(c)
	if err != nil {
		writeValidationError(c)
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
	publicID, err := publicIDFromPath(c)
	if err != nil {
		writeValidationError(c)
		return
	}
	cursor, limit, err := decodeRelationshipPageQuery(c)
	if err != nil {
		writeValidationError(c)
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
	publicID, err := publicIDFromPath(c)
	if err != nil {
		writeValidationError(c)
		return
	}
	cursor, limit, err := decodeRelationshipPageQuery(c)
	if err != nil {
		writeValidationError(c)
		return
	}
	page, err := h.service.ListFollowing(c.Request.Context(), application.ListFollowingQuery{TargetPublicID: publicID, Cursor: cursor, Limit: limit})
	if err != nil {
		writeMappedError(c, err)
		return
	}
	sharedhttp.WriteSuccess(c.Writer, h.relationshipPageResponse(c.Request.Context(), page))
}
