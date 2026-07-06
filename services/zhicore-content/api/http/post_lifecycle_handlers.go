package httpapi

import (
	"strings"
	"time"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) unpublishPost(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := actorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	var req postLifecycleReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.BasePostVersion <= 0 {
		writeValidationError(w)
		return
	}
	result, err := h.service.UnpublishPost(r.Context(), application.PostLifecycleCommand{
		Actor:           actor,
		PostID:          postID,
		BasePostVersion: req.BasePostVersion,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationPostLifecycle)
		return
	}
	sharedhttp.WriteSuccess(w, mapPostLifecycleResponse(result))
}

func (h *Handler) deletePost(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := actorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	basePostVersion, err := optionalPositiveIntQuery(c, "basePostVersion")
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.DeletePost(r.Context(), application.PostLifecycleCommand{
		Actor:           actor,
		PostID:          postID,
		BasePostVersion: int64(basePostVersion),
	})
	if err != nil {
		writeMappedError(w, err, errorOperationPostLifecycle)
		return
	}
	sharedhttp.WriteSuccess(w, mapPostLifecycleResponse(result))
}

func (h *Handler) restorePost(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := actorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	var req postLifecycleReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.BasePostVersion < 0 {
		writeValidationError(w)
		return
	}
	result, err := h.service.RestorePost(r.Context(), application.PostLifecycleCommand{
		Actor:           actor,
		PostID:          postID,
		BasePostVersion: req.BasePostVersion,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationPostLifecycle)
		return
	}
	sharedhttp.WriteSuccess(w, mapPostLifecycleResponse(result))
}

func (h *Handler) schedulePost(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := actorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	var req schedulePostReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	scheduledAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ScheduledAt))
	if err != nil || req.BasePostVersion <= 0 || strings.TrimSpace(req.DraftBodyID) == "" || strings.TrimSpace(req.DraftBodyHash) == "" {
		writeValidationError(w)
		return
	}
	result, err := h.service.SchedulePost(r.Context(), application.SchedulePostCommand{
		Actor:           actor,
		PostID:          postID,
		BasePostVersion: req.BasePostVersion,
		DraftBodyID:     req.DraftBodyID,
		DraftBodyHash:   req.DraftBodyHash,
		ScheduledAt:     scheduledAt.UTC(),
	})
	if err != nil {
		writeMappedError(w, err, errorOperationPostLifecycle)
		return
	}
	sharedhttp.WriteSuccess(w, schedulePostResp{
		PostID:      result.PostID,
		PostVersion: result.PostVersion,
		Status:      result.Status,
		ScheduledAt: formatTime(result.ScheduledAt),
	})
}

func (h *Handler) cancelSchedule(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := actorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	basePostVersion, err := optionalPositiveIntQuery(c, "basePostVersion")
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.CancelSchedule(r.Context(), application.PostLifecycleCommand{
		Actor:           actor,
		PostID:          postID,
		BasePostVersion: int64(basePostVersion),
	})
	if err != nil {
		writeMappedError(w, err, errorOperationPostLifecycle)
		return
	}
	sharedhttp.WriteSuccess(w, mapPostLifecycleResponse(result))
}
