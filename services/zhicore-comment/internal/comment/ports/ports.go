package ports

import (
	"context"
	"errors"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
)

var ErrDependencyUnavailable = errors.New("dependency unavailable")

type CommentablePost struct {
	PostID            domain.PostID
	ContentInternalID domain.ContentInternalID
	AuthorID          domain.UserID
}

type ReplyTarget struct {
	Parent domain.Comment
	Root   domain.Comment
}

type ReplyGuardPreview struct {
	ParentAuthorID domain.UserID
}

type CommentCommandRepository interface {
	FindReplyGuardPreview(ctx context.Context, postID domain.PostID, parentID domain.CommentID) (ReplyGuardPreview, bool, error)
	FindReplyTarget(ctx context.Context, postID domain.PostID, parentID domain.CommentID) (ReplyTarget, error)
	Create(ctx context.Context, draft domain.Comment) (domain.Comment, error)
	InitializeTopLevelRanks(ctx context.Context, comment domain.Comment, now time.Time) error
}

type TopLevelCommentPageQuery struct {
	PostID domain.PostID
	Page   int
	Size   int
	Sort   domain.CommentSort
}

type TopLevelCommentPage struct {
	Items []TopLevelCommentRecord
}

type TopLevelCommentRecord struct {
	Comment domain.Comment
	Stats   domain.CommentStats
}

type CommentQueryRepository interface {
	ListTopLevelComments(ctx context.Context, query TopLevelCommentPageQuery) (TopLevelCommentPage, error)
	BatchGetViewerLiked(ctx context.Context, viewerID domain.UserID, commentIDs []domain.CommentID) (map[domain.CommentID]bool, error)
}

type CommentStatsRepository interface {
	Initialize(ctx context.Context, commentID domain.CommentID, now time.Time) error
	IncrementReplyCount(ctx context.Context, rootID domain.CommentID, now time.Time) error
}

type CommentPostStatsRepository interface {
	IncrementForTopLevel(ctx context.Context, postID domain.PostID, now time.Time) error
	IncrementForReply(ctx context.Context, postID domain.PostID, now time.Time) error
	Get(ctx context.Context, postID domain.PostID) (domain.CommentPostStats, error)
}

type CommentIDCodec interface {
	Encode(id domain.CommentID) domain.PublicCommentID
	Decode(publicID domain.PublicCommentID) (domain.CommentID, error)
}

type ContentPostClient interface {
	CheckPostCommentable(ctx context.Context, postID domain.PostID) (CommentablePost, error)
}

type UserProfileClient interface {
	EnsureUserCanComment(ctx context.Context, userID domain.UserID) error
	BatchGetAuthorSummaries(ctx context.Context, userIDs []domain.UserID) (map[domain.UserID]AuthorSummary, error)
}

type AuthorSummary struct {
	UserID       domain.UserID
	PublicID     string
	DisplayName  string
	AvatarFileID string
	AvatarURL    string
	Unavailable  bool
}

type BlockPair struct {
	BlockerID domain.UserID
	BlockedID domain.UserID
}

type UserRelationClient interface {
	BatchCheckBlocked(ctx context.Context, pairs []BlockPair) (map[BlockPair]bool, error)
}

type CommentMediaReferences struct {
	ImageFileIDs  []string
	VoiceFileID   string
	VoiceDuration int
}

type FileReferenceClient interface {
	EnsureCommentMediaReferenced(ctx context.Context, input CommentMediaReferences) error
}

type CreateCommentRateLimitInput struct {
	ActorUserID domain.UserID
	PostID      domain.PostID
}

type RateLimiter interface {
	AllowCreateComment(ctx context.Context, input CreateCommentRateLimitInput) error
}

type TransactionRunner interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type OutboxMessage struct {
	EventType     string
	AggregateType string
	AggregateID   string
	OccurredAt    time.Time
	Payload       []byte
}

type OutboxPublisher interface {
	Publish(ctx context.Context, message OutboxMessage) error
}

type Clock interface {
	Now() time.Time
}
