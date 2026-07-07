package httpapi

import (
	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) upsertReaderSession(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := optionalActorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	postID, sessionID, err := presencePath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.UpsertReaderSession(r.Context(), application.ReaderSessionCommand{
		Actor:     actor,
		PostID:    postID,
		SessionID: sessionID,
	})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, mapReaderPresenceResponse(result))
}

func (h *Handler) deleteReaderSession(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := optionalActorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	postID, sessionID, err := presencePath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	if err := h.service.DeleteReaderSession(r.Context(), application.ReaderSessionCommand{
		Actor:     actor,
		PostID:    postID,
		SessionID: sessionID,
	}); err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, struct{}{})
}

func (h *Handler) getReaderPresence(c *gin.Context) {
	w, r := c.Writer, c.Request
	postID, err := postIDFromPath(c)
	if err != nil {
		writeValidationError(w)
		return
	}
	result, err := h.service.GetReaderPresence(r.Context(), application.ReaderPresenceQuery{PostID: postID})
	if err != nil {
		writeMappedError(w, err)
		return
	}
	sharedhttp.WriteSuccess(w, mapReaderPresenceResponse(result))
}

func presencePath(c *gin.Context) (string, string, error) {
	postID, err := postIDFromPath(c)
	if err != nil {
		return "", "", err
	}
	sessionID, err := sessionIDFromPath(c)
	if err != nil {
		return "", "", err
	}
	return postID, sessionID, nil
}
