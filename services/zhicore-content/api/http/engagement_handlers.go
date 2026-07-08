package httpapi

import (
	"context"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

type engagementMutationFunc func(context.Context, application.EngagementCommand) (application.EngagementResult, error)

func (h *Handler) likePost(c *gin.Context) {
	h.mutateEngagement(c, h.service.LikePost)
}

func (h *Handler) unlikePost(c *gin.Context) {
	h.mutateEngagement(c, h.service.UnlikePost)
}

func (h *Handler) favoritePost(c *gin.Context) {
	h.mutateEngagement(c, h.service.FavoritePost)
}

func (h *Handler) unfavoritePost(c *gin.Context) {
	h.mutateEngagement(c, h.service.UnfavoritePost)
}

func (h *Handler) mutateEngagement(c *gin.Context, run engagementMutationFunc) {
	w, r := c.Writer, c.Request
	actor, err := actorFromRequest(r)
	if err != nil {
		writeMappedError(w, err, errorOperationEngagement)
		return
	}
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := run(r.Context(), application.EngagementCommand{Actor: actor, PostID: postID})
	if err != nil {
		writeMappedError(w, err, errorOperationEngagement)
		return
	}
	sharedhttp.WriteSuccess(w, mapEngagementMutationResponse(result))
}

func (h *Handler) getPostEngagement(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := optionalActorFromRequest(r)
	if err != nil {
		writeMappedError(w, err, errorOperationEngagement)
		return
	}
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.GetPostEngagement(r.Context(), application.GetPostEngagementQuery{Actor: actor, PostID: postID})
	if err != nil {
		writeMappedError(w, err, errorOperationEngagement)
		return
	}
	sharedhttp.WriteSuccess(w, mapPostEngagementResponse(result))
}

func (h *Handler) batchGetEngagementStatus(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := actorFromRequest(r)
	if err != nil {
		writeMappedError(w, err, errorOperationEngagement)
		return
	}
	var req batchGetEngagementStatusReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if len(req.PostIDs) == 0 || len(req.PostIDs) > 100 {
		writeValidationError(w)
		return
	}
	result, err := h.service.BatchGetEngagementStatus(r.Context(), application.BatchGetEngagementStatusQuery{
		Actor:   actor,
		PostIDs: append([]string(nil), req.PostIDs...),
	})
	if err != nil {
		writeMappedError(w, err, errorOperationEngagement)
		return
	}
	sharedhttp.WriteSuccess(w, mapBatchEngagementStatusResponse(result))
}
