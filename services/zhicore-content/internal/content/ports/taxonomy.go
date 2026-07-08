package ports

import (
	"context"
	"time"
)

type TaxonomyRepository interface {
	ListTags(ctx context.Context, query TagListQuery) ([]TagRecord, error)
	GetTagBySlug(ctx context.Context, slug string) (TagRecord, error)
	SearchTags(ctx context.Context, query TagSearchQuery) ([]TagRecord, error)
	ListHotTags(ctx context.Context, limit int) ([]TagRecord, error)
	ListPublishedPostsByTag(ctx context.Context, query TaggedPostListQuery) ([]PostSummaryRecord, error)
	ListPostTags(ctx context.Context, publicID string) ([]TagRecord, error)
	ReplacePostTags(ctx context.Context, tx Tx, input ReplacePostTagsInput) (PostTagsMutationRecord, error)
	RemovePostTag(ctx context.Context, tx Tx, input RemovePostTagInput) (PostTagsMutationRecord, error)
}

type TagCursor struct {
	Slug string
	ID   int64
}

type TagListQuery struct {
	Cursor TagCursor
	Limit  int
}

type TagSearchQuery struct {
	Query string
	Limit int
}

type TaggedPostListQuery struct {
	Slug   string
	Cursor PublishedPostCursor
	Limit  int
}

type TagRecord struct {
	ID        int64
	PublicID  string
	Name      string
	Slug      string
	PostCount int64
}

type ReplacePostTagsInput struct {
	PostInternalID  int64
	PostPublicID    string
	ActorID         int64
	BasePostVersion int64
	Slugs           []string
	UpdatedAt       time.Time
}

type RemovePostTagInput struct {
	PostInternalID  int64
	PostPublicID    string
	ActorID         int64
	BasePostVersion int64
	Slug            string
	UpdatedAt       time.Time
}

type PostTagsMutationRecord struct {
	PostID      string
	PostVersion int64
	Tags        []TagRecord
	UpdatedAt   time.Time
}
