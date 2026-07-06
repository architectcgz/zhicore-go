package httpapi

import (
	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listPublishedPosts(c *gin.Context) {
	w, r := c.Writer, c.Request
	limit, err := optionalPositiveIntQuery(c, "limit")
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.ListPublishedPosts(r.Context(), application.ListPublishedPostsQuery{
		AuthorID:   c.Query("authorId"),
		Tag:        c.Query("tag"),
		CategoryID: c.Query("categoryId"),
		Cursor:     c.Query("cursor"),
		Limit:      limit,
		Sort:       c.Query("sort"),
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

func (h *Handler) getPostDetail(c *gin.Context) {
	w, r := c.Writer, c.Request
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.GetPostDetail(r.Context(), application.GetPostDetailQuery{PostID: postID})
	if err != nil {
		writeMappedError(w, err, errorOperationPublicPostQuery)
		return
	}
	resp := postDetailResp{Post: mapPostSummaryResponse(result.Post)}
	if result.Body != nil {
		body, ok := mapPostBodyResponse(*result.Body)
		if !ok {
			writeMappedError(w, application.ErrBodySchemaUnsupported, errorOperationPublicPostQuery)
			return
		}
		resp.Body = &body
	}
	sharedhttp.WriteSuccess(w, resp)
}

func (h *Handler) batchGetPublishedPosts(c *gin.Context) {
	w, r := c.Writer, c.Request
	var req batchGetPostsReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if len(req.PostIDs) == 0 || len(req.PostIDs) > 100 {
		writeValidationError(w)
		return
	}
	result, err := h.service.BatchGetPublishedPosts(r.Context(), application.BatchGetPublishedPostsQuery{
		PostIDs: append([]string(nil), req.PostIDs...),
		// includeDeleted is intentionally ignored for anonymous public reads:
		// invisible, deleted and missing posts must collapse into missingPostIds.
		IncludeDeleted: false,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationPublicPostQuery)
		return
	}
	sharedhttp.WriteSuccess(w, batchGetPostsResp{
		Items:          mapPostSummaryResponses(result.Items),
		MissingPostIDs: append([]string(nil), result.MissingPostIDs...),
	})
}

func (h *Handler) getPostBody(c *gin.Context) {
	w, r := c.Writer, c.Request
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}

	result, err := h.service.GetPublishedPostBody(r.Context(), application.GetPublishedPostBodyQuery{PostID: postID})
	if err != nil {
		writeMappedError(w, err, errorOperationGetPostBody)
		return
	}

	blocks, ok := extractCanonicalBlocks(result.CanonicalJSON)
	if !ok {
		// Application owns body validation and repair registration; this guard
		// prevents a corrupted application result from being exposed as a
		// successful empty published body at the HTTP contract boundary.
		writeMappedError(w, application.ErrBodySchemaUnsupported, errorOperationGetPostBody)
		return
	}
	sharedhttp.WriteSuccess(w, postBodyResp{
		BodyID:        result.BodyID,
		SchemaVersion: result.SchemaVersion,
		Format:        "blocks",
		Blocks:        blocks,
		PlainText:     result.PlainText,
		ContentHash:   result.ContentHash,
		SizeBytes:     result.SizeBytes,
		CreatedAt:     formatTime(result.CreatedAt),
	})
}
