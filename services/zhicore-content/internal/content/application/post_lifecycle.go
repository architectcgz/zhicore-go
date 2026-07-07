package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Service) UnpublishPost(ctx context.Context, cmd PostLifecycleCommand) (PostLifecycleResult, error) {
	return s.mutatePostLifecycle(ctx, cmd, lifecycleMutation{
		allowedStatus:  domain.PostStatusPublished,
		targetStatus:   domain.PostStatusDraft,
		rejectStatus:   domain.ErrPostNotPublished,
		reason:         "AUTHOR_UNPUBLISHED",
		publicVisible:  false,
		repositoryCall: s.posts.Unpublish,
	})
}

func (s *Service) DeletePost(ctx context.Context, cmd PostLifecycleCommand) (PostLifecycleResult, error) {
	return s.mutatePostLifecycle(ctx, cmd, lifecycleMutation{
		targetStatus:   domain.PostStatusDeleted,
		reason:         "AUTHOR_DELETED",
		publicVisible:  false,
		repositoryCall: s.posts.DeletePost,
		allowAnyLive:   true,
	})
}

func (s *Service) RestorePost(ctx context.Context, cmd PostLifecycleCommand) (PostLifecycleResult, error) {
	return s.mutatePostLifecycle(ctx, cmd, lifecycleMutation{
		allowedStatus:  domain.PostStatusDeleted,
		targetStatus:   domain.PostStatusDraft,
		rejectStatus:   domain.ErrPostNotFound,
		reason:         "AUTHOR_RESTORED",
		publicVisible:  false,
		repositoryCall: s.posts.RestorePost,
	})
}

type lifecycleMutation struct {
	allowedStatus  domain.PostStatus
	targetStatus   domain.PostStatus
	rejectStatus   error
	reason         string
	publicVisible  bool
	allowAnyLive   bool
	skipOutbox     bool
	repositoryCall func(context.Context, ports.Tx, ports.PostLifecycleUpdate) (ports.PostRecord, error)
}

func (s *Service) mutatePostLifecycle(ctx context.Context, cmd PostLifecycleCommand, mutation lifecycleMutation) (PostLifecycleResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return PostLifecycleResult{}, ErrLoginRequired
	}
	if err := s.enforceRateLimit(ctx, actorRateLimitRequest(ports.RateLimitTypePublishLifecycle, cmd.Actor, cmd.PostID, "post_lifecycle")); err != nil {
		return PostLifecycleResult{}, err
	}

	current, err := s.loadPostForDraftWrite(ctx, cmd.PostID)
	if err != nil {
		return PostLifecycleResult{}, err
	}
	if current.OwnerID != cmd.Actor.UserID {
		return PostLifecycleResult{}, domain.ErrForbidden
	}
	if current.Status == domain.PostStatusDeleted && mutation.allowedStatus != domain.PostStatusDeleted {
		return PostLifecycleResult{}, domain.ErrPostDeleted
	}
	if !mutation.allowAnyLive {
		if current.Status != mutation.allowedStatus {
			if mutation.rejectStatus != nil {
				return PostLifecycleResult{}, mutation.rejectStatus
			}
			return PostLifecycleResult{}, domain.ErrDraftConflict
		}
	} else if current.Status == domain.PostStatusDeleted {
		return PostLifecycleResult{}, domain.ErrPostDeleted
	}

	updatedAt := s.clock.Now()
	var changed ports.PostRecord
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		var err error
		changed, err = mutation.repositoryCall(ctx, tx, ports.PostLifecycleUpdate{
			PublicID:        cmd.PostID,
			OwnerID:         cmd.Actor.UserID,
			BasePostVersion: cmd.BasePostVersion,
			UpdatedAt:       updatedAt,
		})
		if err != nil {
			return err
		}
		if s.outbox == nil || mutation.skipOutbox {
			return nil
		}
		event, err := newPostVisibilityChangedOutboxEvent(
			current,
			changed,
			postVisibilityForStatus(current.Status),
			postVisibilityForStatus(changed.Status),
			mutation.publicVisible,
			mutation.reason,
			updatedAt,
		)
		if err != nil {
			return err
		}
		return s.outbox.Append(ctx, tx, event)
	})
	if err != nil {
		if errors.Is(err, domain.ErrDraftConflict) || errors.Is(err, domain.ErrForbidden) ||
			errors.Is(err, domain.ErrPostDeleted) || errors.Is(err, domain.ErrPostNotFound) ||
			errors.Is(err, domain.ErrPostAlreadyPublished) || errors.Is(err, domain.ErrPostNotPublished) {
			return PostLifecycleResult{}, err
		}
		return PostLifecycleResult{}, fmt.Errorf("%w: mutate post lifecycle", ErrDependencyUnavailable)
	}

	return PostLifecycleResult{
		PostID:      changed.PublicID,
		PostVersion: changed.PostVersion,
		Status:      string(changed.Status),
		UpdatedAt:   updatedAt,
	}, nil
}
