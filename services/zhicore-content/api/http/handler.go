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
	ListTags(ctx context.Context, query application.ListTagsQuery) (application.TagPageResult, error)
	GetTag(ctx context.Context, query application.GetTagQuery) (application.Tag, error)
	SearchTags(ctx context.Context, query application.SearchTagsQuery) ([]application.Tag, error)
	ListHotTags(ctx context.Context, query application.ListHotTagsQuery) ([]application.Tag, error)
	ListPostsByTag(ctx context.Context, query application.ListPostsByTagQuery) (application.ListPublishedPostsResult, error)
	GetPostTags(ctx context.Context, query application.GetPostTagsQuery) ([]application.Tag, error)
	UpdatePostTags(ctx context.Context, command application.UpdatePostTagsCommand) (application.PostTagsMutationResult, error)
	DeletePostTag(ctx context.Context, command application.DeletePostTagCommand) (application.PostTagsMutationResult, error)
	LikePost(ctx context.Context, command application.EngagementCommand) (application.EngagementResult, error)
	UnlikePost(ctx context.Context, command application.EngagementCommand) (application.EngagementResult, error)
	FavoritePost(ctx context.Context, command application.EngagementCommand) (application.EngagementResult, error)
	UnfavoritePost(ctx context.Context, command application.EngagementCommand) (application.EngagementResult, error)
	GetPostEngagement(ctx context.Context, query application.GetPostEngagementQuery) (application.PostEngagementResult, error)
	BatchGetEngagementStatus(ctx context.Context, query application.BatchGetEngagementStatusQuery) (application.BatchEngagementStatusResult, error)
	UpsertReaderSession(ctx context.Context, command application.ReaderSessionCommand) (application.ReaderPresenceResult, error)
	DeleteReaderSession(ctx context.Context, command application.ReaderSessionCommand) error
	GetReaderPresence(ctx context.Context, query application.ReaderPresenceQuery) (application.ReaderPresenceResult, error)
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
	h.router.POST("/api/v1/posts/engagement/batch-status", h.batchGetEngagementStatus)
	h.router.GET("/api/v1/posts/:postId/tags", h.getPostTags)
	h.router.PUT("/api/v1/posts/:postId/tags", h.updatePostTags)
	h.router.DELETE("/api/v1/posts/:postId/tags/:slug", h.deletePostTag)
	h.router.PUT("/api/v1/posts/:postId/like", h.likePost)
	h.router.DELETE("/api/v1/posts/:postId/like", h.unlikePost)
	h.router.PUT("/api/v1/posts/:postId/favorite", h.favoritePost)
	h.router.DELETE("/api/v1/posts/:postId/favorite", h.unfavoritePost)
	h.router.GET("/api/v1/posts/:postId/engagement", h.getPostEngagement)
	h.router.PUT("/api/v1/posts/:postId/reader-sessions/:sessionId", h.upsertReaderSession)
	h.router.DELETE("/api/v1/posts/:postId/reader-sessions/:sessionId", h.deleteReaderSession)
	h.router.GET("/api/v1/posts/:postId/reader-presence", h.getReaderPresence)
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
	h.router.GET("/api/v1/tags", h.listTags)
	h.router.GET("/api/v1/tags/search", h.searchTags)
	h.router.GET("/api/v1/tags/hot", h.listHotTags)
	h.router.GET("/api/v1/tags/:slug/posts", h.listPostsByTag)
	h.router.GET("/api/v1/tags/:slug", h.getTag)
	h.router.GET("/api/v1/admin/content/outbox-events", h.listAdminOutboxEvents)
	h.router.POST("/api/v1/admin/content/outbox-events/:eventId/retry", h.retryAdminOutboxEvent)
}
