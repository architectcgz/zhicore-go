package ports

import (
	"context"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
)

type RelationshipRecord struct {
	ID        int64
	ActorID   domain.UserID
	TargetID  domain.UserID
	Reason    string
	CreatedAt time.Time
}

type RelationshipPage struct {
	Records []RelationshipRecord
	HasMore bool
}

type FollowStats struct {
	FollowersCount int64
	FollowingCount int64
}

type RelationshipRepository interface {
	InsertFollow(ctx context.Context, followerID, followingID domain.UserID, now time.Time) (bool, error)
	DeleteFollow(ctx context.Context, followerID, followingID domain.UserID) (bool, error)
	InsertBlock(ctx context.Context, blockerID, blockedID domain.UserID, reason string, now time.Time) (bool, error)
	DeleteBlock(ctx context.Context, blockerID, blockedID domain.UserID) (bool, error)
	ListBlocked(ctx context.Context, blockerID domain.UserID, cursor string, limit int) (RelationshipPage, error)
	ListFollowers(ctx context.Context, targetID domain.UserID, cursor string, limit int) (RelationshipPage, error)
	ListFollowing(ctx context.Context, targetID domain.UserID, cursor string, limit int) (RelationshipPage, error)
	BatchCheckBlocked(ctx context.Context, pairs []domain.UserPair) (map[domain.UserPair]bool, error)
	CheckFollowing(ctx context.Context, followerID, followingID domain.UserID) (bool, error)
}
