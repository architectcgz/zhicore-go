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
	Profiles ports.ProfileRepository
	Queries  ports.ProfileQueryRepository
	Files    ports.FileReferenceClient
	IDs      ports.PublicIDGenerator
	Outbox   ports.OutboxPublisher
	TxRunner ports.TransactionRunner
	Clock    ports.Clock
	Cache    ports.CacheStore
}

type Service struct {
	profiles ports.ProfileRepository
	queries  ports.ProfileQueryRepository
	files    ports.FileReferenceClient
	ids      ports.PublicIDGenerator
	outbox   ports.OutboxPublisher
	txRunner ports.TransactionRunner
	clock    ports.Clock
	cache    ports.CacheStore
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
	return &Service{
		profiles: deps.Profiles,
		queries:  deps.Queries,
		files:    deps.Files,
		ids:      deps.IDs,
		outbox:   deps.Outbox,
		txRunner: deps.TxRunner,
		clock:    deps.Clock,
		cache:    deps.Cache,
	}, nil
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

func (s *Service) invalidateProfileCache(ctx context.Context, profile domain.Profile) {
	_ = s.cache.Delete(ctx,
		fmt.Sprintf("user:%d:simple", profile.UserID),
		fmt.Sprintf("user:%d:profile", profile.UserID),
		fmt.Sprintf("user:%d:availability", profile.UserID),
		fmt.Sprintf("user:public:%s:id", profile.PublicID),
	)
}
