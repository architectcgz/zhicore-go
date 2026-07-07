package httpapi

import (
	"strings"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listAdminPosts(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := requireAdminActorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	authorID, err := optionalPositiveIntQuery(c, "authorId")
	if err != nil {
		writeValidationError(w)
		return
	}
	page, err := optionalPositiveIntQuery(c, "page")
	if err != nil {
		writeValidationError(w)
		return
	}
	size, err := optionalPositiveIntQuery(c, "size")
	if err != nil {
		writeValidationError(w)
		return
	}

	result, err := h.service.ListAdminPosts(r.Context(), application.ListAdminPostsQuery{
		Actor:    actor,
		Status:   c.Query("status"),
		AuthorID: int64(authorID),
		Page:     page,
		Size:     size,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationAdminPosts)
		return
	}

	items := make([]adminPostResp, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, adminPostResp{
			PostID:             item.PostID,
			AuthorID:           item.AuthorID,
			AuthorName:         item.AuthorName,
			AuthorAvatarFileID: item.AuthorAvatarFileID,
			Title:              item.Title,
			Summary:            item.Summary,
			CoverFileID:        item.CoverFileID,
			Status:             item.Status,
			PostVersion:        item.PostVersion,
			PublishedAt:        formatTime(item.PublishedAt),
			CreatedAt:          formatTime(item.CreatedAt),
			UpdatedAt:          formatTime(item.UpdatedAt),
			Stats:              mapPostStatsResponse(item.Stats),
		})
	}
	sharedhttp.WriteSuccess(w, adminPostListResp{
		Items: items,
		Page:  result.Page,
		Size:  result.Size,
		Total: result.Total,
	})
}

func (h *Handler) deleteAdminPost(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := requireAdminActorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	postID := strings.TrimSpace(c.Param("postId"))
	if postID == "" {
		writeValidationError(w)
		return
	}

	var req adminPostDeleteReq
	if err := decodeOptionalJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	result, err := h.service.DeleteAdminPost(r.Context(), application.DeleteAdminPostCommand{
		Actor:  actor,
		PostID: postID,
		Reason: req.Reason,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationAdminPosts)
		return
	}
	sharedhttp.WriteSuccess(w, adminPostDeleteResp{PostID: result.PostID, Status: result.Status})
}
