package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	userevents "github.com/architectcgz/zhicore-go/libs/contracts/events/user"
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
	AccountID AccountID
	Username  string
}

type UpdateProfileCommand struct {
	UserID                 UserID
	Nickname               *string
	AvatarFileID           *string
	Bio                    *string
	StrangerMessageAllowed *bool
}

type DeactivateUserProfileCommand struct {
	AccountID AccountID
}

type MarkUserDeletedCommand struct {
	UserID     UserID
	OperatorID UserID
	Reason     string
}

type RestoreDeletedUserProfileCommand struct {
	UserID     UserID
	OperatorID UserID
	Reason     string
}

type FollowUserCommand struct {
	ActorUserID    UserID
	TargetPublicID PublicID
}

type UnfollowUserCommand struct {
	ActorUserID    UserID
	TargetPublicID PublicID
}

type BlockUserCommand struct {
	ActorUserID    UserID
	TargetPublicID PublicID
	Reason         string
}

type UnblockUserCommand struct {
	ActorUserID    UserID
	TargetPublicID PublicID
}

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

type RelationshipProfilePage struct {
	Items      []Profile
	NextCursor string
	HasMore    bool
}

const (
	relationshipEventUserFollowed   = userevents.EventFollowed
	relationshipEventUserUnfollowed = userevents.EventUnfollowed
	relationshipEventUserBlocked    = userevents.EventBlocked
	relationshipEventUserUnblocked  = userevents.EventUnblocked
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
	actor, target, err := s.loadActorAndTarget(ctx, domainUserID(cmd.ActorUserID), domainPublicID(cmd.TargetPublicID))
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
	actor, target, err := s.loadActorAndTarget(ctx, domainUserID(cmd.ActorUserID), domainPublicID(cmd.TargetPublicID))
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
	actor, target, err := s.loadActorAndTarget(ctx, domainUserID(cmd.ActorUserID), domainPublicID(cmd.TargetPublicID))
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
	actor, target, err := s.loadActorAndTarget(ctx, domainUserID(cmd.ActorUserID), domainPublicID(cmd.TargetPublicID))
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

func (s *Service) BatchGetUserSimple(ctx context.Context, userIDs []UserID) (BatchUserSimpleResult, error) {
	domainIDs := make([]domain.UserID, 0, len(userIDs))
	for _, userID := range userIDs {
		domainIDs = append(domainIDs, domainUserID(userID))
	}
	profiles, err := s.queries.BatchGetByUserIDs(ctx, domainIDs)
	if err != nil {
		return BatchUserSimpleResult{}, err
	}
	byID := make(map[domain.UserID]domain.Profile, len(profiles))
	for _, profile := range profiles {
		byID[profile.UserID] = profile
	}
	result := BatchUserSimpleResult{Items: make([]UserSimple, 0, len(profiles))}
	seenMissing := map[UserID]bool{}
	for _, userID := range userIDs {
		profile, ok := byID[domainUserID(userID)]
		if !ok {
			// provider 响应按调用方请求顺序返回已命中的用户，同时 missing 列表去重，
			// 方便下游用原请求集合做幂等补偿或降级展示。
			if !seenMissing[userID] {
				result.MissingUserIDs = append(result.MissingUserIDs, userID)
				seenMissing[userID] = true
			}
			continue
		}
		result.Items = append(result.Items, userSimpleFromDomain(profile))
	}
	return result, nil
}

func (s *Service) BatchGetUserAvailability(ctx context.Context, userIDs []UserID) ([]UserAvailability, error) {
	simple, err := s.BatchGetUserSimple(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[UserID]UserSimple, len(simple.Items))
	for _, item := range simple.Items {
		byID[item.UserID] = item
	}
	result := make([]UserAvailability, 0, len(userIDs))
	for _, userID := range userIDs {
		item, ok := byID[userID]
		if !ok {
			// 缺失用户视为不可用于评论互动，但保留 userId 让调用方能定位被拒绝对象。
			result = append(result, UserAvailability{UserID: userID})
			continue
		}
		result = append(result, UserAvailability{
			UserID:    userID,
			Available: item.Status == UserStatusActive,
			Status:    item.Status,
		})
	}
	return result, nil
}

func (s *Service) CheckFollowing(ctx context.Context, followerID, followingID domain.UserID) (bool, error) {
	if err := s.requireRelationshipRepository(); err != nil {
		return false, err
	}
	return s.relationships.CheckFollowing(ctx, followerID, followingID)
}

func (s *Service) CreateProfileForAccount(ctx context.Context, cmd CreateProfileForAccountCommand) (Profile, error) {
	// Auth 注册同步调用这里时，重复 accountId 必须回放同一个 profile，不能再生成新 user。
	existing, err := s.profiles.FindByAccountID(ctx, domain.AccountID(cmd.AccountID))
	if err == nil {
		return profileFromDomain(existing), nil
	}
	if !errors.Is(err, domain.ErrProfileNotFound) {
		return Profile{}, fmt.Errorf("find profile by account id: %w", err)
	}

	publicID, err := s.ids.Generate(ctx)
	if err != nil {
		return Profile{}, fmt.Errorf("generate public id: %w", err)
	}
	now := s.clock.Now()
	profile, err := domain.NewProfileForAccount(domain.AccountID(cmd.AccountID), publicID, cmd.Username, now)
	if err != nil {
		return Profile{}, err
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
		return s.publish(txCtx, userevents.EventProfileCreated, profile.UserID, now, userevents.ProfileCreatedPayload{
			UserID:         int64(profile.UserID),
			AccountID:      int64(profile.AccountID),
			Nickname:       profile.Nickname,
			AvatarFileID:   profile.AvatarFileID,
			ProfileVersion: profile.ProfileVersion,
			OccurredAt:     now,
		})
	}); err != nil {
		return Profile{}, err
	}
	return profileFromDomain(profile), nil
}

func (s *Service) GetMyProfile(ctx context.Context, userID UserID) (Profile, error) {
	profile, err := s.queries.GetByUserID(ctx, domainUserID(userID))
	if err != nil {
		return Profile{}, fmt.Errorf("get my profile: %w", err)
	}
	// DELETED profiles are hidden from owner-facing HTTP just like public profile reads;
	// historical rendering should use dedicated summary APIs, not this editable profile view.
	if profile.Status == domain.UserStatusDeleted {
		return Profile{}, domain.ErrProfileNotFound
	}
	return profileFromDomain(profile), nil
}

func (s *Service) GetUserProfileByPublicID(ctx context.Context, publicID PublicID) (Profile, error) {
	profile, err := s.queries.GetByPublicID(ctx, domainPublicID(publicID))
	if err != nil {
		return Profile{}, fmt.Errorf("get user profile by public id: %w", err)
	}
	if profile.Status == domain.UserStatusDeleted {
		return Profile{}, domain.ErrProfileNotFound
	}
	return profileFromDomain(profile), nil
}

func (s *Service) UpdateProfile(ctx context.Context, cmd UpdateProfileCommand) (Profile, error) {
	if cmd.AvatarFileID != nil && strings.TrimSpace(*cmd.AvatarFileID) != "" {
		avatarFileID := strings.TrimSpace(*cmd.AvatarFileID)
		if err := s.files.EnsureAvatarReferenced(ctx, avatarFileID); err != nil {
			return Profile{}, err
		}
	}

	var updated domain.Profile
	publicChanged := false
	now := s.clock.Now()
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		current, err := s.queries.GetByUserID(txCtx, domainUserID(cmd.UserID))
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
			if err := s.publish(txCtx, userevents.EventProfileUpdated, updated.UserID, now, userevents.ProfileUpdatedPayload{
				UserID:         int64(updated.UserID),
				AccountID:      int64(updated.AccountID),
				Nickname:       updated.Nickname,
				AvatarFileID:   updated.AvatarFileID,
				Bio:            updated.Bio,
				ProfileVersion: updated.ProfileVersion,
				OccurredAt:     now,
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
		return Profile{}, err
	}
	s.invalidateProfileCache(ctx, updated)
	return profileFromDomain(updated), nil
}

func (s *Service) DeactivateUserProfile(ctx context.Context, cmd DeactivateUserProfileCommand) (Profile, error) {
	var (
		updated domain.Profile
		changed bool
		err     error
	)
	now := s.clock.Now()
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		updated, changed, err = s.profiles.DeactivateByAccountID(txCtx, domain.AccountID(cmd.AccountID), now)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		return s.publish(txCtx, userevents.EventDeactivated, updated.UserID, updated.UpdatedAt, userevents.DeactivatedPayload{
			UserID:     int64(updated.UserID),
			AccountID:  int64(updated.AccountID),
			OccurredAt: updated.UpdatedAt,
		})
	}); err != nil {
		return Profile{}, err
	}
	if changed {
		s.invalidateProfileCache(ctx, updated)
	}
	return profileFromDomain(updated), nil
}

func (s *Service) MarkUserDeleted(ctx context.Context, cmd MarkUserDeletedCommand) (Profile, error) {
	var (
		updated domain.Profile
		changed bool
		err     error
	)
	now := s.clock.Now()
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		updated, changed, err = s.profiles.MarkDeleted(txCtx, domainUserID(cmd.UserID), domainUserID(cmd.OperatorID), cmd.Reason, now)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		return s.publish(txCtx, userevents.EventDeleted, updated.UserID, updated.UpdatedAt, userevents.DeletedPayload{
			UserID:     int64(updated.UserID),
			OperatorID: int64(updated.DeletedBy),
			Reason:     updated.DeletedReason,
			OccurredAt: updated.UpdatedAt,
		})
	}); err != nil {
		return Profile{}, err
	}
	if changed {
		s.invalidateProfileCache(ctx, updated)
	}
	return profileFromDomain(updated), nil
}

func (s *Service) RestoreDeletedUserProfile(ctx context.Context, cmd RestoreDeletedUserProfileCommand) (Profile, error) {
	var (
		updated domain.Profile
		changed bool
		err     error
	)
	now := s.clock.Now()
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		updated, changed, err = s.profiles.RestoreDeleted(txCtx, domainUserID(cmd.UserID), domainUserID(cmd.OperatorID), cmd.Reason, now)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		return s.publish(txCtx, userevents.EventRestored, updated.UserID, updated.UpdatedAt, userevents.RestoredPayload{
			UserID:     int64(updated.UserID),
			OperatorID: int64(updated.RestoredBy),
			Reason:     updated.RestoredReason,
			OccurredAt: updated.UpdatedAt,
		})
	}); err != nil {
		return Profile{}, err
	}
	if changed {
		s.invalidateProfileCache(ctx, updated)
	}
	return profileFromDomain(updated), nil
}

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
		return s.publish(ctx, relationshipEventUserFollowed, e.FollowerID, occurredAt, userevents.FollowedPayload{
			FollowerID:  int64(e.FollowerID),
			FollowingID: int64(e.FollowingID),
			OccurredAt:  occurredAt,
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
