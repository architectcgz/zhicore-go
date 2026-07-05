package ports

import (
	"context"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
)

type Tx interface{}

type TransactionRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
}

type PostRepository interface {
	CreateDraft(ctx context.Context, tx Tx, input CreateDraftPost) (PostRecord, error)
	GetForUpdate(ctx context.Context, tx Tx, publicID string) (PostRecord, error)
	SaveDraftBody(ctx context.Context, tx Tx, input SaveDraftBodyUpdate) (PostRecord, error)
	Publish(ctx context.Context, tx Tx, input PublishPostUpdate) (PostRecord, error)
}

type PostQueryRepository interface {
	GetPublishedBodyPointer(ctx context.Context, publicID string) (PublishedBodyPointer, error)
	ListPublishedPosts(ctx context.Context, query PostListQuery) ([]PostSummaryRecord, error)
	GetPublishedPostDetail(ctx context.Context, publicID string) (PostDetailRecord, error)
	BatchGetPublishedPostSummaries(ctx context.Context, publicIDs []string) ([]PostSummaryRecord, error)
}

type BodyReferenceChecker interface {
	IsBodyReferenced(ctx context.Context, bodyID string) (bool, error)
}

type CreateDraftPost struct {
	OwnerID              int64
	OwnerDisplayName     string
	OwnerAvatarFileID    string
	OwnerProfileVersion  int64
	Title                string
	Summary              string
	CoverFileID          string
	TopicID              string
	CategoryID           string
	Tags                 []string
	DraftBodyID          string
	DraftBodyHash        string
	DraftSizeBytes       int
	DraftPlainTextLength int
}

type PostRecord struct {
	ID                    int64
	PublicID              string
	OwnerID               int64
	Status                domain.PostStatus
	PostVersion           int64
	DraftTitle            string
	DraftSummary          string
	DraftCoverFileID      string
	DraftBodyID           string
	DraftBodyHash         string
	DraftSizeBytes        int
	DraftPlainTextLength  int
	PublishedTitle        string
	PublishedSummary      string
	PublishedCoverFileID  string
	PublishedBodyID       string
	PublishedBodyHash     string
	PublishedPlainTextLen int
	PublishedAt           time.Time
}

type SaveDraftBodyUpdate struct {
	PublicID             string
	OwnerID              int64
	BasePostVersion      int64
	BaseDraftBodyID      string
	BaseDraftBodyHash    string
	NewDraftBodyID       string
	NewDraftBodyHash     string
	NewDraftSizeBytes    int
	NewDraftPlainTextLen int
}

type PublishPostUpdate struct {
	PublicID                 string
	OwnerID                  int64
	BasePostVersion          int64
	ExpectedDraftBodyID      string
	ExpectedDraftBodyHash    string
	NewPublishedBodyID       string
	NewPublishedBodyHash     string
	NewPublishedPlainTextLen int
	PublishedAt              time.Time
}

type PublishedBodyPointer struct {
	PostID                int64
	PublicID              string
	Status                domain.PostStatus
	PublishedBodyID       string
	PublishedBodyHash     string
	PublishedPlainTextLen int
}

type PublishedPostCursor struct {
	PublishedAt time.Time
	PublicID    string
}

type PostListQuery struct {
	AuthorID int64
	Cursor   PublishedPostCursor
	Limit    int
}

type PostSummaryRecord struct {
	PostID             string
	AuthorID           int64
	AuthorName         string
	AuthorAvatarFileID string
	Title              string
	Summary            string
	CoverFileID        string
	Status             domain.PostStatus
	PostVersion        int64
	PublishedAt        time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ViewCount          int64
	LikeCount          int64
	FavoriteCount      int64
	CommentCount       int64
}

type PostDetailRecord struct {
	Summary         PostSummaryRecord
	PublishedBodyID string
	PublishedHash   string
}
