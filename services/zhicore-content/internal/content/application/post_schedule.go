package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Service) SchedulePost(ctx context.Context, cmd SchedulePostCommand) (SchedulePostResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return SchedulePostResult{}, ErrLoginRequired
	}
	now := s.clock.Now()
	if !cmd.ScheduledAt.After(now) {
		return SchedulePostResult{}, ErrInvalidArgument
	}
	if err := s.enforceRateLimit(ctx, actorRateLimitRequest(ports.RateLimitTypePublishLifecycle, cmd.Actor, cmd.PostID, "schedule_post")); err != nil {
		return SchedulePostResult{}, err
	}

	current, err := s.loadPostForDraftWrite(ctx, cmd.PostID)
	if err != nil {
		return SchedulePostResult{}, err
	}
	if current.OwnerID != cmd.Actor.UserID {
		return SchedulePostResult{}, domain.ErrForbidden
	}
	if current.Status == domain.PostStatusDeleted {
		return SchedulePostResult{}, domain.ErrPostDeleted
	}
	if current.Status != domain.PostStatusDraft {
		return SchedulePostResult{}, domain.ErrDraftConflict
	}
	if current.PostVersion != cmd.BasePostVersion ||
		current.DraftBodyID != cmd.DraftBodyID ||
		current.DraftBodyHash != cmd.DraftBodyHash {
		return SchedulePostResult{}, domain.ErrDraftConflict
	}

	normalized, err := s.validateDraftForScheduledPublish(ctx, current, cmd)
	if err != nil {
		return SchedulePostResult{}, err
	}
	if s.files != nil {
		if len(normalized.MediaRefs) > 0 {
			if err := s.files.ValidateBodyMediaRefs(ctx, normalized.MediaRefs); err != nil {
				return SchedulePostResult{}, mapFileValidationError(err, ErrMediaRefInvalid)
			}
		}
		if current.DraftCoverFileID != "" {
			if err := s.files.ValidateCoverFile(ctx, current.DraftCoverFileID); err != nil {
				return SchedulePostResult{}, mapFileValidationError(err, ErrCoverUnavailable)
			}
		}
	}

	var scheduled ports.PostRecord
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		var err error
		scheduled, err = s.posts.SchedulePost(ctx, tx, ports.SchedulePostUpdate{
			PublicID:        cmd.PostID,
			OwnerID:         cmd.Actor.UserID,
			BasePostVersion: cmd.BasePostVersion,
			DraftBodyID:     cmd.DraftBodyID,
			DraftBodyHash:   cmd.DraftBodyHash,
			ScheduledAt:     cmd.ScheduledAt,
			UpdatedAt:       now,
		})
		return err
	})
	if err != nil {
		if errors.Is(err, domain.ErrDraftConflict) || errors.Is(err, domain.ErrForbidden) ||
			errors.Is(err, domain.ErrPostDeleted) || errors.Is(err, domain.ErrPostNotFound) {
			return SchedulePostResult{}, err
		}
		return SchedulePostResult{}, fmt.Errorf("%w: schedule post", ErrDependencyUnavailable)
	}
	return SchedulePostResult{
		PostID:      scheduled.PublicID,
		PostVersion: scheduled.PostVersion,
		Status:      string(scheduled.Status),
		ScheduledAt: cmd.ScheduledAt,
	}, nil
}

func (s *Service) CancelSchedule(ctx context.Context, cmd PostLifecycleCommand) (PostLifecycleResult, error) {
	return s.mutatePostLifecycle(ctx, cmd, lifecycleMutation{
		allowedStatus:  domain.PostStatusScheduled,
		targetStatus:   domain.PostStatusDraft,
		rejectStatus:   domain.ErrPostNotPublished,
		reason:         "AUTHOR_CANCELED_SCHEDULE",
		publicVisible:  false,
		repositoryCall: s.posts.CancelSchedule,
		skipOutbox:     true,
	})
}

func (s *Service) validateDraftForScheduledPublish(ctx context.Context, current ports.PostRecord, cmd SchedulePostCommand) (ports.NormalizedBody, error) {
	if current.DraftBodyID == "" || current.DraftBodyHash == "" {
		return ports.NormalizedBody{}, domain.ErrBodyRequired
	}
	draftBody, err := s.bodies.ReadBody(ctx, current.DraftBodyID)
	if err != nil {
		if errors.Is(err, domain.ErrBodyUnavailable) {
			s.appendRepairTask(ctx, ports.BodyRepairTask{
				PostID:       current.ID,
				BodyID:       current.DraftBodyID,
				TaskType:     "draft_body_missing",
				ExpectedHash: current.DraftBodyHash,
				CreatedAt:    s.clock.Now(),
			})
			return ports.NormalizedBody{}, err
		}
		return ports.NormalizedBody{}, fmt.Errorf("%w: read draft body", ErrDependencyUnavailable)
	}
	if draftBody.ContentHash != current.DraftBodyHash || draftBody.ContentHash != cmd.DraftBodyHash {
		return ports.NormalizedBody{}, domain.ErrBodyInconsistent
	}
	normalized, err := s.validateStoredBody(ctx, draftBody)
	if err != nil {
		return ports.NormalizedBody{}, err
	}
	post, err := domain.HydratePost(domain.HydratePostInput{
		ID:       domain.PostID(current.ID),
		PublicID: domain.PublicPostID(current.PublicID),
		OwnerID:  domain.OwnerID(current.OwnerID),
		Title:    current.DraftTitle,
		Summary:  current.DraftSummary,
		Status:   current.Status,
		DraftBody: &domain.BodyPointer{
			ID:              current.DraftBodyID,
			Hash:            current.DraftBodyHash,
			PlainTextLength: len([]rune(normalized.PlainText)),
			SizeBytes:       normalized.SizeBytes,
		},
	})
	if err != nil {
		return ports.NormalizedBody{}, err
	}
	// Scheduling reserves the exact draft that will later be published, so it
	// must enforce the same user-visible publish guards without creating a
	// MongoDB snapshot or exposing the article publicly yet.
	if err := post.Publish(domain.NewPostPublishPolicy(defaultMinPublishedPlainTextRunes), domain.PublishInput{
		DraftBody:   post.DraftBody(),
		PublishedAt: cmd.ScheduledAt,
	}); err != nil {
		return ports.NormalizedBody{}, err
	}
	return normalized, nil
}
