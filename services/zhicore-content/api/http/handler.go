package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

const (
	userIDHeaderName        = "X-User-Id"
	maxJSONRequestBodyBytes = 512 << 10
)

var errLoginRequired = errors.New("login required")

type Service interface {
	CreatePost(ctx context.Context, cmd application.CreatePostCommand) (application.CreatePostResult, error)
	SaveDraftBody(ctx context.Context, cmd application.SaveDraftBodyCommand) (application.SaveDraftBodyResult, error)
	PublishPost(ctx context.Context, cmd application.PublishPostCommand) (application.PublishPostResult, error)
	GetPublishedPostBody(ctx context.Context, query application.GetPublishedPostBodyQuery) (application.GetPublishedPostBodyResult, error)
}

type Handler struct {
	service Service
	router  *gin.Engine
}

func NewHandler(service Service) *gin.Engine {
	h := &Handler{service: service, router: gin.New()}
	h.routes()
	return h.router
}

func (h *Handler) routes() {
	h.router.POST("/api/v1/posts", h.createPost)
	h.router.PUT("/api/v1/posts/:postId/draft/body", h.saveDraftBody)
	h.router.POST("/api/v1/posts/:postId/publish", h.publishPost)
	h.router.GET("/api/v1/posts/:postId/body", h.getPostBody)
}

func (h *Handler) createPost(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}

	var req createPostReq
	if !decodeJSONBody(w, r, &req) {
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	postID, ok := postIDFromPath(w, c)
	if !ok {
		return
	}

	var req saveDraftBodyReq
	if !decodeJSONBody(w, r, &req) {
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	postID, ok := postIDFromPath(w, c)
	if !ok {
		return
	}

	var req publishPostReq
	if !decodeJSONBody(w, r, &req) {
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

func (h *Handler) getPostBody(c *gin.Context) {
	w, r := c.Writer, c.Request
	postID, ok := postIDFromPath(w, c)
	if !ok {
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

type createPostReq struct {
	Title       string       `json:"title"`
	Summary     string       `json:"summary"`
	CoverFileID string       `json:"coverFileId"`
	TopicID     string       `json:"topicId"`
	CategoryID  string       `json:"categoryId"`
	Tags        []string     `json:"tags"`
	Body        *postBodyReq `json:"body"`
}

type postBodyReq struct {
	SchemaVersion int                `json:"schemaVersion"`
	Blocks        application.Blocks `json:"blocks"`
}

type saveDraftBodyReq struct {
	BasePostVersion   int64              `json:"basePostVersion"`
	BaseDraftBodyID   string             `json:"baseDraftBodyId"`
	BaseDraftBodyHash string             `json:"baseDraftBodyHash"`
	SchemaVersion     int                `json:"schemaVersion"`
	Blocks            application.Blocks `json:"blocks"`
	ClientSavedAt     string             `json:"clientSavedAt"`
}

type publishPostReq struct {
	BasePostVersion int64  `json:"basePostVersion"`
	DraftBodyID     string `json:"draftBodyId"`
	DraftBodyHash   string `json:"draftBodyHash"`
}

type createPostResp struct {
	PostID      string `json:"postId"`
	PostVersion int64  `json:"postVersion"`
}

type saveDraftBodyResp struct {
	PostID        string `json:"postId"`
	PostVersion   int64  `json:"postVersion"`
	DraftBodyID   string `json:"draftBodyId"`
	DraftBodyHash string `json:"draftBodyHash"`
	SavedAt       string `json:"savedAt"`
	WordCount     int    `json:"wordCount"`
}

type publishPostResp struct {
	PostID      string `json:"postId"`
	PostVersion int64  `json:"postVersion"`
	PublishedAt string `json:"publishedAt"`
}

type postBodyResp struct {
	BodyID        string          `json:"bodyId"`
	SchemaVersion int             `json:"schemaVersion"`
	Format        string          `json:"format"`
	Blocks        json.RawMessage `json:"blocks"`
	PlainText     string          `json:"plainText"`
	ContentHash   string          `json:"contentHash"`
	SizeBytes     int             `json:"sizeBytes"`
	CreatedAt     string          `json:"createdAt"`
}

func actorFromRequest(r *http.Request) (*application.Actor, bool) {
	raw := strings.TrimSpace(r.Header.Get(userIDHeaderName))
	if raw == "" {
		return nil, false
	}
	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || userID <= 0 {
		return nil, false
	}
	return &application.Actor{UserID: userID}, true
}

func postIDFromPath(w http.ResponseWriter, c *gin.Context) (string, bool) {
	postID := strings.TrimSpace(c.Param("postId"))
	if postID == "" {
		writeValidationError(w)
		return "", false
	}
	return postID, true
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		if isRequestBodyTooLarge(err) {
			sharedhttp.WriteErrorCode(w, http.StatusRequestEntityTooLarge, 4015, "Body too large")
			return false
		}
		writeValidationError(w)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if isRequestBodyTooLarge(err) {
			sharedhttp.WriteErrorCode(w, http.StatusRequestEntityTooLarge, 4015, "Body too large")
			return false
		}
		writeValidationError(w)
		return false
	}
	return true
}

func isRequestBodyTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}

func writeValidationError(w http.ResponseWriter) {
	sharedhttp.WriteErrorCode(w, http.StatusBadRequest, 1001, "Invalid request")
}

type errorOperation string

const (
	errorOperationCreatePost    errorOperation = "createPost"
	errorOperationSaveDraftBody errorOperation = "saveDraftBody"
	errorOperationPublishPost   errorOperation = "publishPost"
	errorOperationGetPostBody   errorOperation = "getPostBody"
)

func writeMappedError(w http.ResponseWriter, err error, operation ...errorOperation) {
	op := errorOperation("")
	if len(operation) > 0 {
		op = operation[0]
	}
	status, code, message, details := errorMapping(err, op)
	opts := make([]sharedhttp.ErrorOption, 0, 1)
	if len(details) > 0 {
		opts = append(opts, sharedhttp.WithDetails(details))
	}
	sharedhttp.WriteErrorCode(w, status, code, message, opts...)
}

func errorMapping(err error, operation errorOperation) (int, int, string, []sharedhttp.ErrorDetail) {
	var validationErr *application.BodyValidationError
	if errors.As(err, &validationErr) {
		status, code, message := bodyValidationMapping(validationErr)
		return status, code, message, validationDetails(validationErr.Details)
	}

	switch {
	case errors.Is(err, errLoginRequired), errors.Is(err, application.ErrLoginRequired):
		return http.StatusUnauthorized, 2006, "Authentication required", nil
	case errors.Is(err, application.ErrInvalidArgument):
		return http.StatusBadRequest, 1001, "Invalid request", nil
	case errors.Is(err, application.ErrDependencyUnavailable):
		return http.StatusServiceUnavailable, 1004, "Service unavailable", nil
	case errors.Is(err, application.ErrBodySchemaUnsupported):
		if operation == errorOperationSaveDraftBody || operation == errorOperationCreatePost {
			return http.StatusBadRequest, 4024, "Body schema unsupported", nil
		}
		return http.StatusInternalServerError, 4024, "Body schema unsupported", nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return http.StatusServiceUnavailable, 1004, "Service unavailable", nil
	case errors.Is(err, application.ErrPostNotFound):
		return http.StatusNotFound, 4001, "Post not found", nil
	case errors.Is(err, application.ErrForbidden):
		return http.StatusForbidden, 2008, "Forbidden", nil
	case errors.Is(err, application.ErrPostAlreadyPublished):
		return http.StatusConflict, 4002, "Post already published", nil
	case errors.Is(err, application.ErrPostDeleted):
		return http.StatusConflict, 4004, "Post deleted", nil
	case errors.Is(err, application.ErrTitleRequired):
		return http.StatusBadRequest, 4005, "Post title is required", nil
	case errors.Is(err, application.ErrBodyRequired):
		return http.StatusBadRequest, 4006, "Post body is required", nil
	case errors.Is(err, application.ErrTitleTooLong):
		return http.StatusBadRequest, 4007, "Post title is too long", nil
	case errors.Is(err, application.ErrBodyTooShort):
		return http.StatusBadRequest, 4016, "Post body text is too short", nil
	case errors.Is(err, application.ErrDraftConflict):
		return http.StatusConflict, 4017, "Draft conflict", nil
	case errors.Is(err, application.ErrBodyUnavailable):
		return http.StatusInternalServerError, 4018, "Body unavailable", nil
	case errors.Is(err, application.ErrBodyInconsistent):
		return http.StatusConflict, 4019, "Body inconsistent", nil
	default:
		return http.StatusInternalServerError, 1000, "Internal server error", nil
	}
}

func extractCanonicalBlocks(canonicalJSON []byte) (json.RawMessage, bool) {
	if len(canonicalJSON) == 0 {
		return nil, false
	}
	var body struct {
		Blocks json.RawMessage `json:"blocks"`
	}
	if err := json.Unmarshal(canonicalJSON, &body); err != nil || len(body.Blocks) == 0 {
		return nil, false
	}
	return body.Blocks, true
}

func bodyValidationMapping(err *application.BodyValidationError) (int, int, string) {
	if err.Truncated {
		return http.StatusBadRequest, 4022, "Too many validation errors"
	}
	for _, detail := range err.Details {
		switch detail.Code {
		case "BODY_TOO_LARGE", "BODY_TEXT_TOO_LONG", "BODY_BLOCK_COUNT_EXCEEDED", "BODY_INLINE_NODE_COUNT_EXCEEDED", "BODY_EXTERNAL_LINK_COUNT_EXCEEDED":
			return http.StatusBadRequest, 4015, "Body too large"
		case "BODY_SCHEMA_UNSUPPORTED":
			return http.StatusBadRequest, 4024, "Body schema unsupported"
		case "BLOCK_TYPE_NOT_ENABLED":
			return http.StatusBadRequest, 4014, "Block type not enabled"
		case "MEDIA_REF_INVALID":
			return http.StatusBadRequest, 4021, "Media reference invalid"
		case "EXTERNAL_EMBED_PROVIDER_NOT_ALLOWED":
			return http.StatusBadRequest, 4020, "External embed provider not allowed"
		}
	}
	return http.StatusBadRequest, 4013, "Body schema invalid"
}

func validationDetails(details []application.ValidationDetail) []sharedhttp.ErrorDetail {
	if len(details) == 0 {
		return nil
	}
	mapped := make([]sharedhttp.ErrorDetail, 0, len(details))
	for _, detail := range details {
		mapped = append(mapped, sharedhttp.ErrorDetail{Path: detail.Path, Code: detail.Code})
	}
	return mapped
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
