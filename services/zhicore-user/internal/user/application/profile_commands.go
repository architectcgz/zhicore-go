package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	userevents "github.com/architectcgz/zhicore-go/libs/contracts/events/user"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
)

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
