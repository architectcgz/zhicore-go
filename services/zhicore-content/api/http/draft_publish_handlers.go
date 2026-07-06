package httpapi

import (
	"strings"
	"time"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) createPost(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := actorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}

	var req createPostReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	var body *application.PostBodyInput
	if req.Body != nil {
		body = &application.PostBodyInput{SchemaVersion: req.Body.SchemaVersion, Blocks: req.Body.Blocks}
	}
	result, err := h.service.CreatePost(r.Context(), application.CreatePostCommand{
		Actor:       actor,
		Title:       req.Title,
		Summary:     req.Summary,
		CoverFileID: req.CoverFileID,
		TopicID:     req.TopicID,
		CategoryID:  req.CategoryID,
		Tags:        append([]string(nil), req.Tags...),
		Body:        body,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationCreatePost)
		return
	}

	sharedhttp.WriteSuccess(w, createPostResp{PostID: result.PostID, PostVersion: result.PostVersion})
}

func (h *Handler) saveDraftBody(c *gin.Context) {
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

	var req saveDraftBodyReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.BasePostVersion <= 0 || req.SchemaVersion <= 0 {
		writeValidationError(w)
		return
	}
	if strings.TrimSpace(req.ClientSavedAt) != "" {
		if _, err := time.Parse(time.RFC3339, req.ClientSavedAt); err != nil {
			writeValidationError(w)
			return
		}
	}

	result, err := h.service.SaveDraftBody(r.Context(), application.SaveDraftBodyCommand{
		Actor:             actor,
		PostID:            postID,
		BasePostVersion:   req.BasePostVersion,
		BaseDraftBodyID:   req.BaseDraftBodyID,
		BaseDraftBodyHash: req.BaseDraftBodyHash,
		Body: application.PostBodyInput{
			SchemaVersion: req.SchemaVersion,
			Blocks:        req.Blocks,
		},
	})
	if err != nil {
		writeMappedError(w, err, errorOperationSaveDraftBody)
		return
	}

	sharedhttp.WriteSuccess(w, saveDraftBodyResp{
		PostID:        result.PostID,
		PostVersion:   result.PostVersion,
		DraftBodyID:   result.DraftBodyID,
		DraftBodyHash: result.DraftBodyHash,
		SavedAt:       formatTime(result.SavedAt),
		WordCount:     result.WordCount,
	})
}

func (h *Handler) publishPost(c *gin.Context) {
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

	var req publishPostReq
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.BasePostVersion <= 0 || strings.TrimSpace(req.DraftBodyID) == "" || strings.TrimSpace(req.DraftBodyHash) == "" {
		writeValidationError(w)
		return
	}

	result, err := h.service.PublishPost(r.Context(), application.PublishPostCommand{
		Actor:           actor,
		PostID:          postID,
		BasePostVersion: req.BasePostVersion,
		DraftBodyID:     req.DraftBodyID,
		DraftBodyHash:   req.DraftBodyHash,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationPublishPost)
		return
	}

	sharedhttp.WriteSuccess(w, publishPostResp{
		PostID:      result.PostID,
		PostVersion: result.PostVersion,
		PublishedAt: formatTime(result.PublishedAt),
	})
}
