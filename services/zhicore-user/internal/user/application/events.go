package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	userevents "github.com/architectcgz/zhicore-go/libs/contracts/events/user"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/ports"
)

func (s *Service) publish(ctx context.Context, eventType string, userID domain.UserID, occurredAt time.Time, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", eventType, err)
	}
	return s.outbox.Publish(ctx, ports.OutboxMessage{
		EventType:     eventType,
		AggregateType: "user",
		AggregateID:   strconv.FormatInt(int64(userID), 10),
		OccurredAt:    occurredAt,
		Payload:       body,
	})
}

func (s *Service) publishRelationshipEvent(ctx context.Context, event domain.RelationshipEvent, occurredAt time.Time) error {
	// Domain events only state the relationship fact. The application layer owns
	// the outward integration event name and JSON payload that enter outbox.
	switch e := event.(type) {
	case domain.UserFollowed:
		actor, err := s.queries.GetByUserID(ctx, e.FollowerID)
		if err != nil {
			return err
		}
		target, err := s.queries.GetByUserID(ctx, e.FollowingID)
		if err != nil {
			return err
		}
		return s.publish(ctx, relationshipEventUserFollowed, e.FollowerID, occurredAt, userevents.FollowedPayload{
			FollowerID:  int64(e.FollowerID),
			FollowingID: int64(e.FollowingID),
			// AvatarFileID is a storage reference, not a browser-safe URL. Omit it
			// until the user service owns a URL resolver for this event snapshot.
			Actor:          userevents.ProfileSnapshot{PublicID: string(actor.PublicID), DisplayName: actor.Nickname},
			TargetPublicID: string(target.PublicID),
			OccurredAt:     occurredAt,
		})
	case domain.UserUnfollowed:
		return s.publish(ctx, relationshipEventUserUnfollowed, e.FollowerID, occurredAt, userevents.UnfollowedPayload{
			FollowerID:  int64(e.FollowerID),
			FollowingID: int64(e.FollowingID),
			Reason:      string(e.Reason),
			OccurredAt:  occurredAt,
		})
	case domain.UserBlocked:
		return s.publish(ctx, relationshipEventUserBlocked, e.BlockerID, occurredAt, userevents.BlockedPayload{
			BlockerID:  int64(e.BlockerID),
			BlockedID:  int64(e.BlockedID),
			Reason:     e.Reason,
			OccurredAt: occurredAt,
		})
	case domain.UserUnblocked:
		return s.publish(ctx, relationshipEventUserUnblocked, e.BlockerID, occurredAt, userevents.UnblockedPayload{
			BlockerID:  int64(e.BlockerID),
			BlockedID:  int64(e.BlockedID),
			OccurredAt: occurredAt,
		})
	default:
		return fmt.Errorf("unknown relationship event %T", event)
	}
}
