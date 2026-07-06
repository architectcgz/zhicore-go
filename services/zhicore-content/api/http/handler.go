package httpapi

import (
	"context"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	"github.com/gin-gonic/gin"
)

const (
	userIDHeaderName        = "X-User-Id"
	userRolesHeaderName     = "X-User-Roles"
	maxJSONRequestBodyBytes = 512 << 10
)

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
