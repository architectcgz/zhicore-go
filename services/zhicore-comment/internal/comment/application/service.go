package application

import (
	"errors"
	"fmt"

	commentevents "github.com/architectcgz/zhicore-go/libs/contracts/events/comment"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

const (
	commentCreatedEventType = commentevents.EventCommentCreated
	commentDeletedEventType = commentevents.EventCommentDeleted
	commentLikedEventType   = commentevents.EventCommentLiked
	commentUnlikedEventType = commentevents.EventCommentUnliked
)

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrDependencyUnavailable  = ports.ErrDependencyUnavailable
	ErrPostNotFound           = ports.ErrPostNotFound
	ErrCommentNotFound        = domain.ErrCommentNotFound
	ErrForbidden              = errors.New("forbidden")
	ErrInteractionBlocked     = ports.ErrInteractionBlocked
	ErrCommentContentRequired = domain.ErrCommentContentRequired
	ErrCommentContentTooLong  = domain.ErrCommentContentTooLong
	ErrParentCommentNotFound  = domain.ErrParentCommentNotFound
	ErrRootCommentNotFound    = domain.ErrRootCommentNotFound
	ErrCommentIDInvalid       = domain.ErrCommentIDInvalid
)

type UserID int64

type PostID string

type PublicCommentID string

type CommentStatus string

type CommentSort string

type DeletedByRole string

const (
	CommentStatusNormal  CommentStatus = "NORMAL"
	CommentStatusDeleted CommentStatus = "DELETED"

	CommentSortRecommended CommentSort = "RECOMMENDED"
	CommentSortHot         CommentSort = "HOT"
	CommentSortTime        CommentSort = "TIME"

	DeletedByRoleAuthor DeletedByRole = "AUTHOR"
	DeletedByRoleAdmin  DeletedByRole = "ADMIN"
)

type Dependencies struct {
	Commands      ports.CommentCommandRepository
	Queries       ports.CommentQueryRepository
	Stats         ports.CommentStatsRepository
	PostStats     ports.CommentPostStatsRepository
	ContentPosts  ports.ContentPostClient
	UserProfiles  ports.UserProfileClient
	UserRelations ports.UserRelationClient
	Files         ports.FileReferenceClient
	IDs           ports.CommentIDCodec
	RateLimiter   ports.RateLimiter
	TxRunner      ports.TransactionRunner
	Outbox        ports.OutboxPublisher
	Clock         ports.Clock
}

type Service struct {
	commands      ports.CommentCommandRepository
	queries       ports.CommentQueryRepository
	stats         ports.CommentStatsRepository
	postStats     ports.CommentPostStatsRepository
	contentPosts  ports.ContentPostClient
	userProfiles  ports.UserProfileClient
	userRelations ports.UserRelationClient
	files         ports.FileReferenceClient
	ids           ports.CommentIDCodec
	rateLimiter   ports.RateLimiter
	txRunner      ports.TransactionRunner
	outbox        ports.OutboxPublisher
	clock         ports.Clock
}

func NewService(deps Dependencies) (*Service, error) {
	for _, item := range []struct {
		name  string
		value any
	}{
		{"Commands", deps.Commands},
		{"Queries", deps.Queries},
		{"Stats", deps.Stats},
		{"PostStats", deps.PostStats},
		{"ContentPosts", deps.ContentPosts},
		{"UserProfiles", deps.UserProfiles},
		{"UserRelations", deps.UserRelations},
		{"Files", deps.Files},
		{"IDs", deps.IDs},
		{"RateLimiter", deps.RateLimiter},
		{"TxRunner", deps.TxRunner},
		{"Outbox", deps.Outbox},
		{"Clock", deps.Clock},
	} {
		if item.value == nil {
			return nil, fmt.Errorf("%s is required", item.name)
		}
	}
	return &Service{
		commands:      deps.Commands,
		queries:       deps.Queries,
		stats:         deps.Stats,
		postStats:     deps.PostStats,
		contentPosts:  deps.ContentPosts,
		userProfiles:  deps.UserProfiles,
		userRelations: deps.UserRelations,
		files:         deps.Files,
		ids:           deps.IDs,
		rateLimiter:   deps.RateLimiter,
		txRunner:      deps.TxRunner,
		outbox:        deps.Outbox,
		clock:         deps.Clock,
	}, nil
}
