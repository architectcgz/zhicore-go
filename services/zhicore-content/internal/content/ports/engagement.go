package ports

import (
	"context"
	"time"
)

type EngagementAction string

const (
	EngagementActionLike       EngagementAction = "LIKE"
	EngagementActionUnlike     EngagementAction = "UNLIKE"
	EngagementActionFavorite   EngagementAction = "FAVORITE"
	EngagementActionUnfavorite EngagementAction = "UNFAVORITE"
)

type EngagementRepository interface {
	MutateEngagement(ctx context.Context, tx Tx, input EngagementMutationInput) (EngagementMutationRecord, error)
	GetPostEngagement(ctx context.Context, postID string) (PostEngagementRecord, error)
	BatchGetViewerStatus(ctx context.Context, userID int64, postIDs []string) ([]EngagementStatusRecord, error)
}

type EngagementCacheStore interface {
	BatchGetViewerStatus(ctx context.Context, userID int64, postIDs []string) ([]EngagementStatusRecord, error)
	StoreMutation(ctx context.Context, record EngagementMutationRecord) error
	StoreViewerStatus(ctx context.Context, userID int64, records []EngagementStatusRecord) error
}

type EngagementMutationInput struct {
	PostID     string
	ActorID    int64
	Action     EngagementAction
	OccurredAt time.Time
}

type EngagementMutationRecord struct {
	PostInternalID   int64
	PostID           string
	AuthorID         int64
	ActorID          int64
	Changed          bool
	Liked            bool
	Favorited        bool
	AggregateVersion int64
	Stats            PostStatsRecord
}

type PostEngagementRecord struct {
	PostID string
	Stats  PostStatsRecord
}

type EngagementStatusRecord struct {
	PostID    string
	Liked     bool
	Favorited bool
}

type PostStatsRecord struct {
	ViewCount     int64
	LikeCount     int64
	FavoriteCount int64
	CommentCount  int64
}
