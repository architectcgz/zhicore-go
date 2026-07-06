package application

import (
	"context"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
)

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
