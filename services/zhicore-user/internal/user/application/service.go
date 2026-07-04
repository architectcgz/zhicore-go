package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/ports"
)

type Dependencies struct {
	Profiles      ports.ProfileRepository
	Queries       ports.ProfileQueryRepository
	Relationships ports.RelationshipRepository
	Files         ports.FileReferenceClient
	IDs           ports.PublicIDGenerator
	Outbox        ports.OutboxPublisher
	TxRunner      ports.TransactionRunner
	Clock         ports.Clock
	Cache         ports.CacheStore
	CacheFailures ports.CacheFailureRecorder
}

type Service struct {
	profiles      ports.ProfileRepository
	queries       ports.ProfileQueryRepository
	relationships ports.RelationshipRepository
	files         ports.FileReferenceClient
	ids           ports.PublicIDGenerator
	outbox        ports.OutboxPublisher
	txRunner      ports.TransactionRunner
	clock         ports.Clock
	cache         ports.CacheStore
	cacheFailures ports.CacheFailureRecorder
}

type CreateProfileForAccountCommand struct {
	AccountID domain.AccountID
	Username  string
}

type UpdateProfileCommand struct {
	UserID                 domain.UserID
	Nickname               *string
	AvatarFileID           *string
	Bio                    *string
	StrangerMessageAllowed *bool
}

type DeactivateUserProfileCommand struct {
	AccountID domain.AccountID
}

type MarkUserDeletedCommand struct {
	UserID     domain.UserID
	OperatorID domain.UserID
	Reason     string
}

type RestoreDeletedUserProfileCommand struct {
	UserID     domain.UserID
	OperatorID domain.UserID
	Reason     string
}

type FollowUserCommand struct {
	ActorUserID    domain.UserID
	TargetPublicID domain.PublicID
}

type UnfollowUserCommand struct {
	ActorUserID    domain.UserID
	TargetPublicID domain.PublicID
}

type BlockUserCommand struct {
	ActorUserID    domain.UserID
	TargetPublicID domain.PublicID
	Reason         string
}

type UnblockUserCommand struct {
	ActorUserID    domain.UserID
	TargetPublicID domain.PublicID
}

type ListBlockedUsersQuery struct {
	ActorUserID domain.UserID
	Cursor      string
	Limit       int
}

type ListFollowersQuery struct {
	TargetPublicID domain.PublicID
	Cursor         string
	Limit          int
}

type ListFollowingQuery struct {
	TargetPublicID domain.PublicID
	Cursor         string
	Limit          int
}

type RelationshipProfilePage struct {
	Items      []domain.Profile
	NextCursor string
	HasMore    bool
}

const (
	relationshipEventUserFollowed   = "user.followed"
	relationshipEventUserUnfollowed = "user.unfollowed"
	relationshipEventUserBlocked    = "user.blocked"
	relationshipEventUserUnblocked  = "user.unblocked"
)

func NewService(deps Dependencies) (*Service, error) {
	if deps.Profiles == nil {
		return nil, fmt.Errorf("Profiles is required")
	}
	if deps.Queries == nil {
		return nil, fmt.Errorf("Queries is required")
	}
	if deps.Files == nil {
		return nil, fmt.Errorf("Files is required")
	}
	if deps.IDs == nil {
		return nil, fmt.Errorf("IDs is required")
	}
	if deps.Outbox == nil {
		return nil, fmt.Errorf("Outbox is required")
	}
	if deps.TxRunner == nil {
		return nil, fmt.Errorf("TxRunner is required")
	}
	if deps.Clock == nil {
		return nil, fmt.Errorf("Clock is required")
	}
	if deps.Cache == nil {
		return nil, fmt.Errorf("Cache is required")
	}
	if deps.CacheFailures == nil {
		return nil, fmt.Errorf("CacheFailures is required")
	}
	return &Service{
		profiles:      deps.Profiles,
		queries:       deps.Queries,
		relationships: deps.Relationships,
		files:         deps.Files,
		ids:           deps.IDs,
		outbox:        deps.Outbox,
		txRunner:      deps.TxRunner,
		clock:         deps.Clock,
		cache:         deps.Cache,
		cacheFailures: deps.CacheFailures,
	}, nil
}

func (s *Service) FollowUser(ctx context.Context, cmd FollowUserCommand) error {
	if err := s.requireRelationshipRepository(); err != nil {
		return err
	}
	actor, target, err := s.loadActorAndTarget(ctx, cmd.ActorUserID, cmd.TargetPublicID)
	if err != nil {
		return err
	}
	blocked, err := s.anyDirectionBlocked(ctx, actor.UserID, target.UserID)
	if err != nil {
		return err
	}
	plan, err := domain.PlanFollow(actor, target, blocked)
	if err != nil {
		return err
	}

	now := s.clock.Now()
	return s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		created, err := s.relationships.InsertFollow(txCtx, plan.FollowerID, plan.FollowingID, now)
		if err != nil || !created {
			return err
		}
		return s.publishRelationshipEvent(txCtx, plan.Event(), now)
	})
}

func (s *Service) UnfollowUser(ctx context.Context, cmd UnfollowUserCommand) error {
	if err := s.requireRelationshipRepository(); err != nil {
		return err
	}
	actor, target, err := s.loadActorAndTarget(ctx, cmd.ActorUserID, cmd.TargetPublicID)
	if err != nil {
		return err
	}
	plan, err := domain.PlanUnfollow(actor, target)
	if err != nil {
		return err
	}

	now := s.clock.Now()
	return s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		deleted, err := s.relationships.DeleteFollow(txCtx, plan.FollowerID, plan.FollowingID)
		if err != nil || !deleted {
			return err
		}
		return s.publishRelationshipEvent(txCtx, plan.Event(), now)
	})
}

func (s *Service) BlockUser(ctx context.Context, cmd BlockUserCommand) error {
	if err := s.requireRelationshipRepository(); err != nil {
		return err
	}
	actor, target, err := s.loadActorAndTarget(ctx, cmd.ActorUserID, cmd.TargetPublicID)
	if err != nil {
		return err
	}
	plan, err := domain.PlanBlock(actor, target, cmd.Reason)
	if err != nil {
		return err
	}

	now := s.clock.Now()
	return s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		created, err := s.relationships.InsertBlock(txCtx, plan.BlockerID, plan.BlockedID, plan.Reason, now)
		if err != nil || !created {
			return err
		}
		if err := s.publishRelationshipEvent(txCtx, plan.Event(), now); err != nil {
			return err
		}
		for _, pair := range plan.RemovedFollows {
			if err := s.deleteFollowForBlock(txCtx, pair, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) UnblockUser(ctx context.Context, cmd UnblockUserCommand) error {
	if err := s.requireRelationshipRepository(); err != nil {
		return err
	}
	actor, target, err := s.loadActorAndTarget(ctx, cmd.ActorUserID, cmd.TargetPublicID)
	if err != nil {
		return err
	}
	plan, err := domain.PlanUnblock(actor, target)
	if err != nil {
		return err
	}

	now := s.clock.Now()
	return s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		deleted, err := s.relationships.DeleteBlock(txCtx, plan.BlockerID, plan.BlockedID)
		if err != nil || !deleted {
			return err
		}
		return s.publishRelationshipEvent(txCtx, plan.Event(), now)
	})
}

func (s *Service) ListBlockedUsers(ctx context.Context, query ListBlockedUsersQuery) (RelationshipProfilePage, error) {
	if err := s.requireRelationshipRepository(); err != nil {
		return RelationshipProfilePage{}, err
	}
	actor, err := s.queries.GetByUserID(ctx, query.ActorUserID)
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
	target, err := s.loadVisiblePublicProfile(ctx, query.TargetPublicID)
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
	target, err := s.loadVisiblePublicProfile(ctx, query.TargetPublicID)
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

func (s *Service) BatchCheckBlocked(ctx context.Context, pairs []domain.UserPair) (map[domain.UserPair]bool, error) {
	if err := s.requireRelationshipRepository(); err != nil {
		return nil, err
	}
	return s.relationships.BatchCheckBlocked(ctx, pairs)
}

func (s *Service) CheckFollowing(ctx context.Context, followerID, followingID domain.UserID) (bool, error) {
	if err := s.requireRelationshipRepository(); err != nil {
		return false, err
	}
	return s.relationships.CheckFollowing(ctx, followerID, followingID)
}

func (s *Service) CreateProfileForAccount(ctx context.Context, cmd CreateProfileForAccountCommand) (domain.Profile, error) {
	// Auth 注册同步调用这里时，重复 accountId 必须回放同一个 profile，不能再生成新 user。
	existing, err := s.profiles.FindByAccountID(ctx, cmd.AccountID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domain.ErrProfileNotFound) {
		return domain.Profile{}, fmt.Errorf("find profile by account id: %w", err)
	}

	publicID, err := s.ids.Generate(ctx)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("generate public id: %w", err)
	}
	now := s.clock.Now()
	profile, err := domain.NewProfileForAccount(cmd.AccountID, publicID, cmd.Username, now)
	if err != nil {
		return domain.Profile{}, err
	}

	created := false
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		profile, created, err = s.profiles.CreateOrGetByAccountID(txCtx, profile)
		if err != nil {
			return err
		}
		if !created {
			return nil
		}
		return s.publish(txCtx, "user.profile.created", profile.UserID, now, map[string]any{
			"userId":         profile.UserID,
			"accountId":      profile.AccountID,
			"nickname":       profile.Nickname,
			"avatarFileId":   profile.AvatarFileID,
			"profileVersion": profile.ProfileVersion,
			"occurredAt":     now,
		})
	}); err != nil {
		return domain.Profile{}, err
	}
	return profile, nil
}

func (s *Service) GetMyProfile(ctx context.Context, userID domain.UserID) (domain.Profile, error) {
	profile, err := s.queries.GetByUserID(ctx, userID)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("get my profile: %w", err)
	}
	// DELETED profiles are hidden from owner-facing HTTP just like public profile reads;
	// historical rendering should use dedicated summary APIs, not this editable profile view.
	if profile.Status == domain.UserStatusDeleted {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	return profile, nil
}

func (s *Service) GetUserProfileByPublicID(ctx context.Context, publicID domain.PublicID) (domain.Profile, error) {
	profile, err := s.queries.GetByPublicID(ctx, publicID)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("get user profile by public id: %w", err)
	}
	if profile.Status == domain.UserStatusDeleted {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	return profile, nil
}

func (s *Service) UpdateProfile(ctx context.Context, cmd UpdateProfileCommand) (domain.Profile, error) {
	if cmd.AvatarFileID != nil && strings.TrimSpace(*cmd.AvatarFileID) != "" {
		avatarFileID := strings.TrimSpace(*cmd.AvatarFileID)
		if err := s.files.EnsureAvatarReferenced(ctx, avatarFileID); err != nil {
			return domain.Profile{}, err
		}
	}

	var updated domain.Profile
	publicChanged := false
	now := s.clock.Now()
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		current, err := s.queries.GetByUserID(txCtx, cmd.UserID)
		if err != nil {
			return fmt.Errorf("load profile for update: %w", err)
		}
		// A DELETED profile is intentionally indistinguishable from missing for edit flows.
		// DEACTIVATED remains a separate non-active state and maps to USER_NOT_ACTIVE.
		if current.Status == domain.UserStatusDeleted {
			return domain.ErrProfileNotFound
		}

		patch := domain.ProfileUpdate{
			Nickname:               current.Nickname,
			AvatarFileID:           current.AvatarFileID,
			Bio:                    current.Bio,
			StrangerMessageAllowed: current.StrangerMessageAllowed,
		}
		if cmd.Nickname != nil {
			patch.Nickname = *cmd.Nickname
		}
		if cmd.AvatarFileID != nil {
			patch.AvatarFileID = *cmd.AvatarFileID
		}
		if cmd.Bio != nil {
			patch.Bio = *cmd.Bio
		}
		if cmd.StrangerMessageAllowed != nil {
			patch.StrangerMessageAllowed = *cmd.StrangerMessageAllowed
		}

		updated, publicChanged, err = current.ApplyUpdate(patch, now)
		if err != nil {
			return err
		}
		if publicChanged {
			// profileVersion 只能由 repository 在事务内原子递增并回传，application 不能自行推算版本。
			updated, err = s.profiles.UpdatePublicProfile(txCtx, updated)
			if err != nil {
				return err
			}
			if err := s.publish(txCtx, "user.profile.updated", updated.UserID, now, map[string]any{
				"userId":         updated.UserID,
				"accountId":      updated.AccountID,
				"nickname":       updated.Nickname,
				"avatarFileId":   updated.AvatarFileID,
				"bio":            updated.Bio,
				"profileVersion": updated.ProfileVersion,
				"occurredAt":     now,
			}); err != nil {
				return err
			}
			return nil
		}
		updated, err = s.profiles.Update(txCtx, updated)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return domain.Profile{}, err
	}
	s.invalidateProfileCache(ctx, updated)
	return updated, nil
}

func (s *Service) DeactivateUserProfile(ctx context.Context, cmd DeactivateUserProfileCommand) (domain.Profile, error) {
	var (
		updated domain.Profile
		changed bool
		err     error
	)
	now := s.clock.Now()
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		updated, changed, err = s.profiles.DeactivateByAccountID(txCtx, cmd.AccountID, now)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		return s.publish(txCtx, "user.deactivated", updated.UserID, updated.UpdatedAt, map[string]any{
			"userId":     updated.UserID,
			"accountId":  updated.AccountID,
			"occurredAt": updated.UpdatedAt,
		})
	}); err != nil {
		return domain.Profile{}, err
	}
	if changed {
		s.invalidateProfileCache(ctx, updated)
	}
	return updated, nil
}

func (s *Service) MarkUserDeleted(ctx context.Context, cmd MarkUserDeletedCommand) (domain.Profile, error) {
	var (
		updated domain.Profile
		changed bool
		err     error
	)
	now := s.clock.Now()
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		updated, changed, err = s.profiles.MarkDeleted(txCtx, cmd.UserID, cmd.OperatorID, cmd.Reason, now)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		return s.publish(txCtx, "user.deleted", updated.UserID, updated.UpdatedAt, map[string]any{
			"userId":     updated.UserID,
			"operatorId": updated.DeletedBy,
			"reason":     updated.DeletedReason,
			"occurredAt": updated.UpdatedAt,
		})
	}); err != nil {
		return domain.Profile{}, err
	}
	if changed {
		s.invalidateProfileCache(ctx, updated)
	}
	return updated, nil
}

func (s *Service) RestoreDeletedUserProfile(ctx context.Context, cmd RestoreDeletedUserProfileCommand) (domain.Profile, error) {
	var (
		updated domain.Profile
		changed bool
		err     error
	)
	now := s.clock.Now()
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		updated, changed, err = s.profiles.RestoreDeleted(txCtx, cmd.UserID, cmd.OperatorID, cmd.Reason, now)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		return s.publish(txCtx, "user.restored", updated.UserID, updated.UpdatedAt, map[string]any{
			"userId":     updated.UserID,
			"operatorId": updated.RestoredBy,
			"reason":     updated.RestoredReason,
			"occurredAt": updated.UpdatedAt,
		})
	}); err != nil {
		return domain.Profile{}, err
	}
	if changed {
		s.invalidateProfileCache(ctx, updated)
	}
	return updated, nil
}

func (s *Service) publish(ctx context.Context, eventType string, userID domain.UserID, occurredAt time.Time, payload map[string]any) error {
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

func (s *Service) publishRelationshipEvent(ctx context.Context, event domain.RelationshipEvent, occurredAt time.Time) error {
	// Domain events only state the relationship fact. The application layer owns
	// the outward integration event name and JSON payload that enter outbox.
	switch e := event.(type) {
	case domain.UserFollowed:
		return s.publish(ctx, relationshipEventUserFollowed, e.FollowerID, occurredAt, map[string]any{
			"followerId":  e.FollowerID,
			"followingId": e.FollowingID,
			"occurredAt":  occurredAt,
		})
	case domain.UserUnfollowed:
		return s.publish(ctx, relationshipEventUserUnfollowed, e.FollowerID, occurredAt, map[string]any{
			"followerId":  e.FollowerID,
			"followingId": e.FollowingID,
			"reason":      string(e.Reason),
			"occurredAt":  occurredAt,
		})
	case domain.UserBlocked:
		return s.publish(ctx, relationshipEventUserBlocked, e.BlockerID, occurredAt, map[string]any{
			"blockerId":  e.BlockerID,
			"blockedId":  e.BlockedID,
			"reason":     e.Reason,
			"occurredAt": occurredAt,
		})
	case domain.UserUnblocked:
		return s.publish(ctx, relationshipEventUserUnblocked, e.BlockerID, occurredAt, map[string]any{
			"blockerId":  e.BlockerID,
			"blockedId":  e.BlockedID,
			"occurredAt": occurredAt,
		})
	default:
		return fmt.Errorf("unknown relationship event %T", event)
	}
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
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    page.HasMore,
	}, nil
}

func (s *Service) invalidateProfileCache(ctx context.Context, profile domain.Profile) {
	keys := []string{
		fmt.Sprintf("user:%d:simple", profile.UserID),
		fmt.Sprintf("user:%d:profile", profile.UserID),
		fmt.Sprintf("user:%d:availability", profile.UserID),
		fmt.Sprintf("user:public:%s:id", profile.PublicID),
	}
	if err := s.cache.Delete(ctx, keys...); err != nil {
		s.cacheFailures.RecordCacheDeleteFailure(ctx, "user.profile.invalidate", keys, err)
	}
}
