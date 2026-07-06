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
	userRolesHeaderName     = "X-User-Roles"
	maxJSONRequestBodyBytes = 512 << 10
)

var errLoginRequired = errors.New("login required")

type Service interface {
	CreatePost(ctx context.Context, cmd application.CreatePostCommand) (application.CreatePostResult, error)
	SaveDraftBody(ctx context.Context, cmd application.SaveDraftBodyCommand) (application.SaveDraftBodyResult, error)
	PublishPost(ctx context.Context, cmd application.PublishPostCommand) (application.PublishPostResult, error)
	UnpublishPost(ctx context.Context, cmd application.PostLifecycleCommand) (application.PostLifecycleResult, error)
	DeletePost(ctx context.Context, cmd application.PostLifecycleCommand) (application.PostLifecycleResult, error)
	RestorePost(ctx context.Context, cmd application.PostLifecycleCommand) (application.PostLifecycleResult, error)
	SchedulePost(ctx context.Context, cmd application.SchedulePostCommand) (application.SchedulePostResult, error)
	CancelSchedule(ctx context.Context, cmd application.PostLifecycleCommand) (application.PostLifecycleResult, error)
	GetPublishedPostBody(ctx context.Context, query application.GetPublishedPostBodyQuery) (application.GetPublishedPostBodyResult, error)
	ListPublishedPosts(ctx context.Context, query application.ListPublishedPostsQuery) (application.ListPublishedPostsResult, error)
	GetPostDetail(ctx context.Context, query application.GetPostDetailQuery) (application.GetPostDetailResult, error)
	BatchGetPublishedPosts(ctx context.Context, query application.BatchGetPublishedPostsQuery) (application.BatchGetPublishedPostsResult, error)
	ListAuthorPosts(ctx context.Context, query application.ListAuthorPostsQuery) (application.AuthorPostPageResult, error)
	ListAuthorDrafts(ctx context.Context, query application.ListAuthorDraftsQuery) (application.AuthorPostPageResult, error)
	GetAuthorDraft(ctx context.Context, query application.GetAuthorDraftQuery) (application.AuthorDraftResult, error)
	UpdateDraftMeta(ctx context.Context, command application.UpdateDraftMetaCommand) (application.DraftMutationResult, error)
	DeleteAuthorDraft(ctx context.Context, command application.DeleteAuthorDraftCommand) (application.DraftMutationResult, error)
	ListAdminOutboxEvents(ctx context.Context, query application.ListAdminOutboxEventsQuery) (application.ListAdminOutboxEventsResult, error)
	RetryAdminOutboxEvent(ctx context.Context, command application.RetryAdminOutboxEventCommand) (application.RetryAdminOutboxEventResult, error)
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
	h.router.GET("/api/v1/posts", h.listPublishedPosts)
	h.router.POST("/api/v1/posts/batch-get", h.batchGetPublishedPosts)
	h.router.GET("/api/v1/me/posts", h.listAuthorPosts)
	h.router.GET("/api/v1/me/drafts", h.listAuthorDrafts)
	h.router.GET("/api/v1/posts/:postId", h.getPostDetail)
	h.router.GET("/api/v1/posts/:postId/draft", h.getAuthorDraft)
	h.router.PATCH("/api/v1/posts/:postId/draft/meta", h.updateDraftMeta)
	h.router.DELETE("/api/v1/posts/:postId/draft", h.deleteAuthorDraft)
	h.router.PUT("/api/v1/posts/:postId/draft/body", h.saveDraftBody)
	h.router.POST("/api/v1/posts/:postId/publish", h.publishPost)
	h.router.POST("/api/v1/posts/:postId/unpublish", h.unpublishPost)
	h.router.POST("/api/v1/posts/:postId/schedule", h.schedulePost)
	h.router.DELETE("/api/v1/posts/:postId/schedule", h.cancelSchedule)
	h.router.POST("/api/v1/posts/:postId/restore", h.restorePost)
	h.router.DELETE("/api/v1/posts/:postId", h.deletePost)
	h.router.GET("/api/v1/posts/:postId/body", h.getPostBody)
	h.router.GET("/api/v1/admin/content/outbox-events", h.listAdminOutboxEvents)
	h.router.POST("/api/v1/admin/content/outbox-events/:eventId/retry", h.retryAdminOutboxEvent)
}

func (h *Handler) listAuthorPosts(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	limit, ok := optionalPositiveIntQuery(w, c, "limit")
	if !ok {
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	limit, ok := optionalPositiveIntQuery(w, c, "limit")
	if !ok {
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	postID, ok := postIDFromPath(w, c)
	if !ok {
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	postID, ok := postIDFromPath(w, c)
	if !ok {
		return
	}
	var req updateDraftMetaReq
	if !decodeJSONBody(w, r, &req) {
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	postID, ok := postIDFromPath(w, c)
	if !ok {
		return
	}
	result, err := h.service.DeleteAuthorDraft(r.Context(), application.DeleteAuthorDraftCommand{Actor: actor, PostID: postID})
	if err != nil {
		writeMappedError(w, err, errorOperationAuthorWorkbench)
		return
	}
	sharedhttp.WriteSuccess(w, mapDraftMutationResponse(result))
}

func (h *Handler) listPublishedPosts(c *gin.Context) {
	w, r := c.Writer, c.Request
	limit, ok := optionalPositiveIntQuery(w, c, "limit")
	if !ok {
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
	postID, ok := postIDFromPath(w, c)
	if !ok {
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
	if !decodeJSONBody(w, r, &req) {
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

func (h *Handler) unpublishPost(c *gin.Context) {
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
	var req postLifecycleReq
	if !decodeJSONBody(w, r, &req) {
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	postID, ok := postIDFromPath(w, c)
	if !ok {
		return
	}
	basePostVersion, ok := optionalPositiveIntQuery(w, c, "basePostVersion")
	if !ok {
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	postID, ok := postIDFromPath(w, c)
	if !ok {
		return
	}
	var req postLifecycleReq
	if !decodeJSONBody(w, r, &req) {
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	postID, ok := postIDFromPath(w, c)
	if !ok {
		return
	}
	var req schedulePostReq
	if !decodeJSONBody(w, r, &req) {
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
	actor, ok := actorFromRequest(r)
	if !ok {
		writeMappedError(w, errLoginRequired)
		return
	}
	postID, ok := postIDFromPath(w, c)
	if !ok {
		return
	}
	basePostVersion, ok := optionalPositiveIntQuery(w, c, "basePostVersion")
	if !ok {
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

func (h *Handler) listAdminOutboxEvents(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := requireAdminActorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	page, ok := optionalPositiveIntQuery(w, c, "page")
	if !ok {
		return
	}
	size, ok := optionalPositiveIntQuery(w, c, "size")
	if !ok {
		return
	}

	result, err := h.service.ListAdminOutboxEvents(r.Context(), application.ListAdminOutboxEventsQuery{
		Actor:     actor,
		Status:    c.Query("status"),
		EventType: c.Query("eventType"),
		Page:      page,
		Size:      size,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationAdminOutbox)
		return
	}

	items := make([]adminOutboxEventResp, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, adminOutboxEventResp{
			EventID:          item.EventID,
			EventType:        item.EventType,
			AggregateType:    item.AggregateType,
			AggregateID:      item.AggregateID,
			AggregateVersion: item.AggregateVersion,
			Status:           item.Status,
			RetryCount:       item.RetryCount,
			LastError:        item.LastError,
			OccurredAt:       formatTime(item.OccurredAt),
			CreatedAt:        formatTime(item.CreatedAt),
			UpdatedAt:        formatTime(item.UpdatedAt),
		})
	}
	sharedhttp.WriteSuccess(w, adminOutboxListResp{
		Items: items,
		Page:  result.Page,
		Size:  result.Size,
		Total: result.Total,
	})
}

func (h *Handler) retryAdminOutboxEvent(c *gin.Context) {
	w, r := c.Writer, c.Request
	actor, err := requireAdminActorFromRequest(r)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	eventID := strings.TrimSpace(c.Param("eventId"))
	if eventID == "" {
		writeValidationError(w)
		return
	}

	var req adminOutboxRetryReq
	if !decodeJSONBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		writeValidationError(w)
		return
	}

	result, err := h.service.RetryAdminOutboxEvent(r.Context(), application.RetryAdminOutboxEventCommand{
		Actor:   actor,
		EventID: eventID,
		Reason:  req.Reason,
	})
	if err != nil {
		writeMappedError(w, err, errorOperationAdminOutbox)
		return
	}
	sharedhttp.WriteSuccess(w, adminOutboxRetryResp{
		EventID:    result.EventID,
		Status:     result.Status,
		RetryCount: result.RetryCount,
		RetriedAt:  formatTime(result.RetriedAt),
	})
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
	return &application.Actor{UserID: userID, Roles: rolesFromRequest(r)}, true
}

func requireAdminActorFromRequest(r *http.Request) (*application.Actor, error) {
	actor, ok := actorFromRequest(r)
	if !ok {
		return nil, errLoginRequired
	}
	if !actor.HasRole("admin") && !actor.HasRole("role_admin") {
		return nil, application.ErrRoleRequired
	}
	return actor, nil
}

func rolesFromRequest(r *http.Request) []string {
	raw := strings.TrimSpace(r.Header.Get(userRolesHeaderName))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	roles := make([]string, 0, len(parts))
	for _, part := range parts {
		if role := strings.TrimSpace(part); role != "" {
			roles = append(roles, role)
		}
	}
	return roles
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

func optionalPositiveIntQuery(w http.ResponseWriter, c *gin.Context, key string) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		writeValidationError(w)
		return 0, false
	}
	return value, true
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
	errorOperationCreatePost      errorOperation = "createPost"
	errorOperationSaveDraftBody   errorOperation = "saveDraftBody"
	errorOperationPublishPost     errorOperation = "publishPost"
	errorOperationPostLifecycle   errorOperation = "postLifecycle"
	errorOperationGetPostBody     errorOperation = "getPostBody"
	errorOperationPublicPostQuery errorOperation = "publicPostQuery"
	errorOperationAuthorWorkbench errorOperation = "authorWorkbench"
	errorOperationAdminOutbox     errorOperation = "adminOutbox"
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
	case errors.Is(err, application.ErrTaxonomyReferenceNotFound):
		return http.StatusNotFound, 4012, "Category not found", nil
	case errors.Is(err, application.ErrMediaRefInvalid):
		return http.StatusBadRequest, 4021, "Media reference invalid", nil
	case errors.Is(err, application.ErrCoverUnavailable):
		return http.StatusBadRequest, 4023, "Cover unavailable", nil
	case errors.Is(err, application.ErrDependencyUnavailable):
		return http.StatusServiceUnavailable, 1004, "Service unavailable", nil
	case errors.Is(err, application.ErrRoleRequired):
		return http.StatusForbidden, 2007, "Role required", nil
	case errors.Is(err, application.ErrBodySchemaUnsupported):
		if operation == errorOperationSaveDraftBody || operation == errorOperationCreatePost {
			return http.StatusBadRequest, 4024, "Body schema unsupported", nil
		}
		return http.StatusInternalServerError, 4024, "Body schema unsupported", nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return http.StatusServiceUnavailable, 1004, "Service unavailable", nil
	case errors.Is(err, application.ErrPostNotFound):
		return http.StatusNotFound, 4001, "Post not found", nil
	case errors.Is(err, application.ErrOutboxEventNotFound):
		return http.StatusNotFound, 1005, "Data not found", nil
	case errors.Is(err, application.ErrForbidden):
		return http.StatusForbidden, 2008, "Forbidden", nil
	case errors.Is(err, application.ErrPostAlreadyPublished):
		return http.StatusConflict, 4002, "Post already published", nil
	case errors.Is(err, application.ErrPostNotPublished):
		return http.StatusConflict, 4003, "Post not published", nil
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

func mapPostSummaryResponses(items []application.PostSummary) []postSummaryResp {
	resp := make([]postSummaryResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, mapPostSummaryResponse(item))
	}
	return resp
}

func mapPostSummaryResponse(item application.PostSummary) postSummaryResp {
	return postSummaryResp{
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
		Stats: postStatsResp{
			ViewCount:     item.Stats.ViewCount,
			LikeCount:     item.Stats.LikeCount,
			FavoriteCount: item.Stats.FavoriteCount,
			CommentCount:  item.Stats.CommentCount,
		},
	}
}

func writePostPage(w http.ResponseWriter, result application.AuthorPostPageResult) {
	sharedhttp.WriteSuccess(w, cursorPageResp[postSummaryResp]{
		Items:      mapPostSummaryResponses(result.Items),
		NextCursor: result.NextCursor,
		HasMore:    result.HasMore,
		Limit:      result.Limit,
	})
}

func mapAuthorDraftResponse(item application.AuthorDraftResult) authorDraftResp {
	resp := authorDraftResp{
		PostID:        item.PostID,
		PostVersion:   item.PostVersion,
		Title:         item.Title,
		Summary:       item.Summary,
		CoverFileID:   item.CoverFileID,
		Status:        item.Status,
		DraftBodyID:   item.DraftBodyID,
		DraftBodyHash: item.DraftBodyHash,
		CreatedAt:     formatTime(item.CreatedAt),
		UpdatedAt:     formatTime(item.UpdatedAt),
	}
	if item.Body != nil {
		body, ok := mapPostBodyResponse(*item.Body)
		if ok {
			resp.Body = &body
		}
	}
	return resp
}

func mapPostLifecycleResponse(result application.PostLifecycleResult) postLifecycleResp {
	return postLifecycleResp{
		PostID:      result.PostID,
		PostVersion: result.PostVersion,
		Status:      result.Status,
		UpdatedAt:   formatTime(result.UpdatedAt),
	}
}

func mapDraftMutationResponse(item application.DraftMutationResult) draftMutationResp {
	return draftMutationResp{
		PostID:      item.PostID,
		PostVersion: item.PostVersion,
		Title:       item.Title,
		Summary:     item.Summary,
		CoverFileID: item.CoverFileID,
		UpdatedAt:   formatTime(item.UpdatedAt),
	}
}

func mapPostBodyResponse(body application.PostBodyResult) (postBodyResp, bool) {
	blocks, ok := extractCanonicalBlocks(body.CanonicalJSON)
	if !ok {
		return postBodyResp{}, false
	}
	return postBodyResp{
		BodyID:        body.BodyID,
		SchemaVersion: body.SchemaVersion,
		Format:        "blocks",
		Blocks:        blocks,
		PlainText:     body.PlainText,
		ContentHash:   body.ContentHash,
		SizeBytes:     body.SizeBytes,
		CreatedAt:     formatTime(body.CreatedAt),
	}, true
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
