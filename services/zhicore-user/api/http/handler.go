package httpapi

import (
	"context"

	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
	"github.com/gin-gonic/gin"
)

const userIDHeaderName = "X-User-Id"

type Service interface {
	GetMyProfile(ctx context.Context, userID application.UserID) (application.Profile, error)
	GetUserProfileByPublicID(ctx context.Context, publicID application.PublicID) (application.Profile, error)
	UpdateProfile(ctx context.Context, cmd application.UpdateProfileCommand) (application.Profile, error)
	BlockUser(ctx context.Context, cmd application.BlockUserCommand) error
	UnblockUser(ctx context.Context, cmd application.UnblockUserCommand) error
	ListBlockedUsers(ctx context.Context, query application.ListBlockedUsersQuery) (application.RelationshipProfilePage, error)
	FollowUser(ctx context.Context, cmd application.FollowUserCommand) error
	UnfollowUser(ctx context.Context, cmd application.UnfollowUserCommand) error
	ListFollowers(ctx context.Context, query application.ListFollowersQuery) (application.RelationshipProfilePage, error)
	ListFollowing(ctx context.Context, query application.ListFollowingQuery) (application.RelationshipProfilePage, error)
	BatchGetUserSimple(ctx context.Context, userIDs []application.UserID) (application.BatchUserSimpleResult, error)
	BatchGetUserAvailability(ctx context.Context, userIDs []application.UserID) ([]application.UserAvailability, error)
	BatchCheckBlocked(ctx context.Context, pairs []application.UserPair) (map[application.UserPair]bool, error)
	ListFollowerShard(ctx context.Context, query application.ListFollowerShardQuery) (application.FollowerShardPage, error)
}

type AvatarURLResolver interface {
	ResolveAvatarURL(ctx context.Context, fileID string) (string, error)
}

type Handler struct {
	service  Service
	resolver AvatarURLResolver
	router   *gin.Engine
}

func NewHandler(service Service, resolver AvatarURLResolver) *gin.Engine {
	h := &Handler{
		service:  service,
		resolver: resolver,
		router:   gin.New(),
	}
	h.routes()
	return h.router
}

func (h *Handler) routes() {
	h.router.GET("/api/v1/users/me", h.getMe)
	h.router.GET("/api/v1/users/me/blocked", h.listBlockedUsers)
	h.router.GET("/api/v1/users/:publicId", h.getProfile)
	h.router.PATCH("/api/v1/users/me/profile", h.updateProfile)
	h.router.POST("/api/v1/users/:publicId/block", h.blockUser)
	h.router.DELETE("/api/v1/users/:publicId/block", h.unblockUser)
	h.router.POST("/api/v1/users/:publicId/follow", h.followUser)
	h.router.DELETE("/api/v1/users/:publicId/follow", h.unfollowUser)
	h.router.GET("/api/v1/users/:publicId/followers", h.listFollowers)
	h.router.GET("/api/v1/users/:publicId/following", h.listFollowing)
	h.router.POST(usercontract.BatchAvailabilityPath, h.batchAvailability)
	h.router.POST(usercontract.BatchSimplePath, h.batchSimple)
	h.router.POST(usercontract.BatchCheckBlockedPath, h.batchCheckBlocked)
	h.router.POST(usercontract.ListFollowerShardPath, h.listFollowerShard)
}
