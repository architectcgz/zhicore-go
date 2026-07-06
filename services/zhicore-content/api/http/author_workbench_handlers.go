package httpapi

import (
	"net/http"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listAuthorPosts(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := actorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	limit, err := optionalPositiveIntQuery(c, "limit")
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.ListAuthorPosts(r.Context(), application.ListAuthorPostsQuery{
		Actor:  actor,
		Status: c.Query("status"),
		Cursor: c.Query("cursor"),
		Limit:  limit,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationAuthorWorkbench)
		return
	}
	writePostPage(w, result)
}

func (h *Handler) listAuthorDrafts(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := actorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	limit, err := optionalPositiveIntQuery(c, "limit")
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.ListAuthorDrafts(r.Context(), application.ListAuthorDraftsQuery{
		Actor:  actor,
		Cursor: c.Query("cursor"),
		Limit:  limit,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationAuthorWorkbench)
		return
	}
	writePostPage(w, result)
}

func (h *Handler) getAuthorDraft(c *gin.Context) {
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
	result, err := h.service.GetAuthorDraft(r.Context(), application.GetAuthorDraftQuery{Actor: actor, PostID: postID})
	if err != nil {
		writeMappedError(w, err, errorOperationAuthorWorkbench)
		return
	}
	resp := mapAuthorDraftResponse(result)
	sharedhttp.WriteSuccess(w, resp)
}

func (h *Handler) updateDraftMeta(c *gin.Context) {
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
	var req updateDraftMetaReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.BasePostVersion <= 0 {
		writeValidationError(w)
		return
	}
	result, err := h.service.UpdateDraftMeta(r.Context(), application.UpdateDraftMetaCommand{
		Actor:           actor,
		PostID:          postID,
		BasePostVersion: req.BasePostVersion,
		Title:           req.Title,
		Summary:         req.Summary,
		CoverFileID:     req.CoverFileID,
		TopicID:         req.TopicID,
		CategoryID:      req.CategoryID,
		Tags:            req.Tags,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationAuthorWorkbench)
		return
	}
	sharedhttp.WriteSuccess(w, mapDraftMutationResponse(result))
}

func (h *Handler) deleteAuthorDraft(c *gin.Context) {
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
	result, err := h.service.DeleteAuthorDraft(r.Context(), application.DeleteAuthorDraftCommand{Actor: actor, PostID: postID})
	if err != nil {
		writeMappedError(w, err, errorOperationAuthorWorkbench)
		return
	}
	sharedhttp.WriteSuccess(w, mapDraftMutationResponse(result))
}

func writePostPage(w http.ResponseWriter, result application.AuthorPostPageResult) {
	sharedhttp.WriteSuccess(w, cursorPageResp[postSummaryResp]{
		Items:      mapPostSummaryResponses(result.Items),
		NextCursor: result.NextCursor,
		HasMore:    result.HasMore,
		Limit:      result.Limit,
	})
}
