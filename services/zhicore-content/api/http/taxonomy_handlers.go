package httpapi

import (
	"strconv"
	"strings"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listTags(c *gin.Context) {
	w, r := c.Writer, c.Request
	limit, err := optionalPositiveIntQuery(c, "limit")
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.ListTags(r.Context(), application.ListTagsQuery{Cursor: c.Query("cursor"), Limit: limit})
	if err != nil {
		writeMappedError(w, err, errorOperationPublicPostQuery)
		return
	}
	sharedhttp.WriteSuccess(w, cursorPageResp[tagResp]{
		Items:      mapTagResponses(result.Items),
		NextCursor: result.NextCursor,
		HasMore:    result.HasMore,
		Limit:      result.Limit,
	})
}

func (h *Handler) getTag(c *gin.Context) {
	w, r := c.Writer, c.Request
	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		writeValidationError(w)
		return
	}
	result, err := h.service.GetTag(r.Context(), application.GetTagQuery{Slug: slug})
	if err != nil {
		writeMappedError(w, err, errorOperationPublicPostQuery)
		return
	}
	sharedhttp.WriteSuccess(w, mapTagResponse(result))
}

func (h *Handler) searchTags(c *gin.Context) {
	w, r := c.Writer, c.Request
	limit, err := optionalPositiveIntQuery(c, "limit")
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.SearchTags(r.Context(), application.SearchTagsQuery{Query: c.Query("q"), Limit: limit})
	if err != nil {
		writeMappedError(w, err, errorOperationPublicPostQuery)
		return
	}
	sharedhttp.WriteSuccess(w, mapTagResponses(result))
}

func (h *Handler) listHotTags(c *gin.Context) {
	w, r := c.Writer, c.Request
	limit, err := optionalPositiveIntQuery(c, "limit")
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.ListHotTags(r.Context(), application.ListHotTagsQuery{Limit: limit})
	if err != nil {
		writeMappedError(w, err, errorOperationPublicPostQuery)
		return
	}
	sharedhttp.WriteSuccess(w, mapTagResponses(result))
}

func (h *Handler) listPostsByTag(c *gin.Context) {
	w, r := c.Writer, c.Request
	limit, err := optionalPositiveIntQuery(c, "limit")
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.ListPostsByTag(r.Context(), application.ListPostsByTagQuery{
		Slug:   c.Param("slug"),
		Cursor: c.Query("cursor"),
		Limit:  limit,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationPublicPostQuery)
		return
	}
	sharedhttp.WriteSuccess(w, cursorPageResp[postSummaryResp]{
		Items:      mapPostSummaryResponses(result.Items),
		NextCursor: result.NextCursor,
		HasMore:    result.HasMore,
		Limit:      result.Limit,
	})
}

func (h *Handler) getPostTags(c *gin.Context) {
	w, r := c.Writer, c.Request
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.GetPostTags(r.Context(), application.GetPostTagsQuery{PostID: postID})
	if err != nil {
		writeMappedError(w, err, errorOperationPublicPostQuery)
		return
	}
	sharedhttp.WriteSuccess(w, mapTagResponses(result))
}

func (h *Handler) updatePostTags(c *gin.Context) {
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
	var req updatePostTagsReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.BasePostVersion <= 0 || req.Tags == nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.UpdatePostTags(r.Context(), application.UpdatePostTagsCommand{
		Actor:           actor,
		PostID:          postID,
		BasePostVersion: req.BasePostVersion,
		Tags:            append([]string(nil), (*req.Tags)...),
	})
	if err != nil {
		writeMappedError(w, err, errorOperationAuthorWorkbench)
		return
	}
	sharedhttp.WriteSuccess(w, mapPostTagsMutationResponse(result))
}

func (h *Handler) deletePostTag(c *gin.Context) {
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
	basePostVersion, err := parseRequiredPositiveInt64(c.Query("basePostVersion"))
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.DeletePostTag(r.Context(), application.DeletePostTagCommand{
		Actor:           actor,
		PostID:          postID,
		BasePostVersion: basePostVersion,
		Slug:            c.Param("slug"),
	})
	if err != nil {
		writeMappedError(w, err, errorOperationAuthorWorkbench)
		return
	}
	sharedhttp.WriteSuccess(w, mapPostTagsMutationResponse(result))
}

func parseRequiredPositiveInt64(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, application.ErrInvalidArgument
	}
	return value, nil
}
