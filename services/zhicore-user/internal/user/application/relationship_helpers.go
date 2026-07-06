package application

import (
	"context"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/ports"
)

func (s *Service) requireRelationshipRepository() error {
	if s.relationships == nil {
		return ports.ErrDependencyUnavailable
	}
	return nil
}

func (s *Service) loadActorAndTarget(ctx context.Context, actorID domain.UserID, targetPublicID domain.PublicID) (domain.Profile, domain.Profile, error) {
	actor, err := s.queries.GetByUserID(ctx, actorID)
	if err != nil {
		return domain.Profile{}, domain.Profile{}, err
	}
	target, err := s.queries.GetByPublicID(ctx, targetPublicID)
	if err != nil {
		return domain.Profile{}, domain.Profile{}, err
	}
	return actor, target, nil
}

func (s *Service) loadVisiblePublicProfile(ctx context.Context, publicID domain.PublicID) (domain.Profile, error) {
	profile, err := s.queries.GetByPublicID(ctx, publicID)
	if err != nil {
		return domain.Profile{}, err
	}
	if profile.Status == domain.UserStatusDeleted {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	return profile, nil
}

func (s *Service) anyDirectionBlocked(ctx context.Context, actorID, targetID domain.UserID) (bool, error) {
	pairs := []domain.UserPair{
		{ActorID: actorID, TargetID: targetID},
		{ActorID: targetID, TargetID: actorID},
	}
	checked, err := s.relationships.BatchCheckBlocked(ctx, pairs)
	if err != nil {
		return false, err
	}
	return checked[pairs[0]] || checked[pairs[1]], nil
}

func (s *Service) deleteFollowForBlock(ctx context.Context, pair domain.UserPair, now time.Time) error {
	deleted, err := s.relationships.DeleteFollow(ctx, pair.ActorID, pair.TargetID)
	if err != nil || !deleted {
		return err
	}
	return s.publishRelationshipEvent(ctx, pair.UnfollowedEvent(domain.UnfollowReasonBlocked), now)
}

func (s *Service) relationshipProfiles(ctx context.Context, page ports.RelationshipPage, userIDForRecord func(ports.RelationshipRecord) domain.UserID) (RelationshipProfilePage, error) {
	items := make([]domain.Profile, 0, len(page.Records))
	for _, record := range page.Records {
		profile, err := s.queries.GetByUserID(ctx, userIDForRecord(record))
		if err != nil {
			return RelationshipProfilePage{}, err
		}
		items = append(items, profile)
	}
	nextCursor := ""
	if page.HasMore && len(page.Records) > 0 {
		nextCursor = domain.EncodeRelationshipCursor(page.Records[len(page.Records)-1].ID)
	}
	return RelationshipProfilePage{
		Items:      profilesFromDomain(items),
		NextCursor: nextCursor,
		HasMore:    page.HasMore,
	}, nil
}
