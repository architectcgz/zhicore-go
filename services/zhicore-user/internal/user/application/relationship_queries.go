package application

import (
	"context"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/ports"
)

type ListBlockedUsersQuery struct {
	ActorUserID UserID
	Cursor      string
	Limit       int
}

type ListFollowersQuery struct {
	TargetPublicID PublicID
	Cursor         string
	Limit          int
}

type ListFollowingQuery struct {
	TargetPublicID PublicID
	Cursor         string
	Limit          int
}

type ListFollowerShardQuery struct {
	FollowingID   UserID
	AudienceClass string
	ActiveSince   string
	Cursor        string
	Limit         int
}

type RelationshipProfilePage struct {
	Items      []Profile
	NextCursor string
	HasMore    bool
}

type FollowerShardPage struct {
	FollowerIDs []UserID
	NextCursor  string
	HasMore     bool
}

func (s *Service) ListBlockedUsers(ctx context.Context, query ListBlockedUsersQuery) (RelationshipProfilePage, error) {
	if err := s.requireRelationshipRepository(); err != nil {
		return RelationshipProfilePage{}, err
	}
	actor, err := s.queries.GetByUserID(ctx, domainUserID(query.ActorUserID))
	if err != nil {
		return RelationshipProfilePage{}, err
	}
	if actor.Status != domain.UserStatusActive {
		return RelationshipProfilePage{}, domain.ErrUserNotActive
	}
	page, err := s.relationships.ListBlocked(ctx, actor.UserID, query.Cursor, domain.NormalizeRelationshipLimit(query.Limit))
	if err != nil {
		return RelationshipProfilePage{}, err
	}
	return s.relationshipProfiles(ctx, page, func(record ports.RelationshipRecord) domain.UserID {
		return record.TargetID
	})
}

func (s *Service) ListFollowers(ctx context.Context, query ListFollowersQuery) (RelationshipProfilePage, error) {
	if err := s.requireRelationshipRepository(); err != nil {
		return RelationshipProfilePage{}, err
	}
	target, err := s.loadVisiblePublicProfile(ctx, domainPublicID(query.TargetPublicID))
	if err != nil {
		return RelationshipProfilePage{}, err
	}
	page, err := s.relationships.ListFollowers(ctx, target.UserID, query.Cursor, domain.NormalizeRelationshipLimit(query.Limit))
	if err != nil {
		return RelationshipProfilePage{}, err
	}
	return s.relationshipProfiles(ctx, page, func(record ports.RelationshipRecord) domain.UserID {
		return record.ActorID
	})
}

func (s *Service) ListFollowing(ctx context.Context, query ListFollowingQuery) (RelationshipProfilePage, error) {
	if err := s.requireRelationshipRepository(); err != nil {
		return RelationshipProfilePage{}, err
	}
	target, err := s.loadVisiblePublicProfile(ctx, domainPublicID(query.TargetPublicID))
	if err != nil {
		return RelationshipProfilePage{}, err
	}
	page, err := s.relationships.ListFollowing(ctx, target.UserID, query.Cursor, domain.NormalizeRelationshipLimit(query.Limit))
	if err != nil {
		return RelationshipProfilePage{}, err
	}
	return s.relationshipProfiles(ctx, page, func(record ports.RelationshipRecord) domain.UserID {
		return record.TargetID
	})
}

func (s *Service) ListFollowerShard(ctx context.Context, query ListFollowerShardQuery) (FollowerShardPage, error) {
	if err := s.requireRelationshipRepository(); err != nil {
		return FollowerShardPage{}, err
	}
	page, err := s.relationships.ListFollowers(ctx, domainUserID(query.FollowingID), query.Cursor, domain.NormalizeRelationshipLimit(query.Limit))
	if err != nil {
		return FollowerShardPage{}, err
	}
	followerIDs := make([]UserID, 0, len(page.Records))
	for _, record := range page.Records {
		// Notification fanout needs only stable follower IDs; resolving profiles here
		// would turn a high-volume shard read into an unnecessary profile hot path.
		followerIDs = append(followerIDs, UserID(record.ActorID))
	}
	nextCursor := ""
	if page.HasMore && len(page.Records) > 0 {
		nextCursor = domain.EncodeRelationshipCursor(page.Records[len(page.Records)-1].ID)
	}
	return FollowerShardPage{
		FollowerIDs: followerIDs,
		NextCursor:  nextCursor,
		HasMore:     page.HasMore,
	}, nil
}

func (s *Service) BatchCheckBlocked(ctx context.Context, pairs []UserPair) (map[UserPair]bool, error) {
	if err := s.requireRelationshipRepository(); err != nil {
		return nil, err
	}
	domainPairs := make([]domain.UserPair, 0, len(pairs))
	for _, pair := range pairs {
		domainPairs = append(domainPairs, domain.UserPair{ActorID: domainUserID(pair.ActorID), TargetID: domainUserID(pair.TargetID)})
	}
	checked, err := s.relationships.BatchCheckBlocked(ctx, domainPairs)
	if err != nil {
		return nil, err
	}
	result := make(map[UserPair]bool, len(pairs))
	for _, pair := range pairs {
		result[pair] = checked[domain.UserPair{ActorID: domainUserID(pair.ActorID), TargetID: domainUserID(pair.TargetID)}]
	}
	return result, nil
}

func (s *Service) CheckFollowing(ctx context.Context, followerID, followingID domain.UserID) (bool, error) {
	if err := s.requireRelationshipRepository(); err != nil {
		return false, err
	}
	return s.relationships.CheckFollowing(ctx, followerID, followingID)
}
