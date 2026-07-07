package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Service) PublishPost(ctx context.Context, cmd PublishPostCommand) (PublishPostResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return PublishPostResult{}, ErrLoginRequired
	}
	if err := s.enforceRateLimit(ctx, actorRateLimitRequest(ports.RateLimitTypePublishLifecycle, cmd.Actor, cmd.PostID, "publish_post")); err != nil {
		return PublishPostResult{}, err
	}

	current, err := s.loadPostForDraftWrite(ctx, cmd.PostID)
	if err != nil {
		return PublishPostResult{}, err
	}
	if current.OwnerID != cmd.Actor.UserID {
		return PublishPostResult{}, domain.ErrForbidden
	}
	if current.Status == domain.PostStatusDeleted {
		return PublishPostResult{}, domain.ErrPostDeleted
	}
	if current.Status == domain.PostStatusPublished {
		return PublishPostResult{}, domain.ErrPostAlreadyPublished
	}
	if current.DraftBodyID == "" || current.DraftBodyHash == "" {
		return PublishPostResult{}, domain.ErrBodyRequired
	}
	if current.PostVersion != cmd.BasePostVersion ||
		current.DraftBodyID != cmd.DraftBodyID ||
		current.DraftBodyHash != cmd.DraftBodyHash {
		return PublishPostResult{}, domain.ErrDraftConflict
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
			return PublishPostResult{}, err
		}
		return PublishPostResult{}, fmt.Errorf("%w: read draft body", ErrDependencyUnavailable)
	}
	if draftBody.ContentHash != current.DraftBodyHash || draftBody.ContentHash != cmd.DraftBodyHash {
		s.appendRepairTask(ctx, ports.BodyRepairTask{
			PostID:       current.ID,
			BodyID:       current.DraftBodyID,
			TaskType:     "body_hash_mismatch",
			ExpectedHash: current.DraftBodyHash,
			ObservedHash: draftBody.ContentHash,
			CreatedAt:    s.clock.Now(),
		})
		return PublishPostResult{}, domain.ErrBodyInconsistent
	}
	normalized, err := s.validateStoredBody(ctx, draftBody)
	if err != nil {
		if errors.Is(err, domain.ErrBodyInconsistent) {
			s.appendRepairTask(ctx, ports.BodyRepairTask{
				PostID:       current.ID,
				BodyID:       current.DraftBodyID,
				TaskType:     "body_hash_mismatch",
				ExpectedHash: current.DraftBodyHash,
				ObservedHash: draftBody.ContentHash,
				CreatedAt:    s.clock.Now(),
			})
		}
		return PublishPostResult{}, err
	}
	if s.files != nil {
		if len(normalized.MediaRefs) > 0 {
			if err := s.files.ValidateBodyMediaRefs(ctx, normalized.MediaRefs); err != nil {
				return PublishPostResult{}, mapFileValidationError(err, ErrMediaRefInvalid)
			}
		}
		if current.DraftCoverFileID != "" {
			if err := s.files.ValidateCoverFile(ctx, current.DraftCoverFileID); err != nil {
				return PublishPostResult{}, mapFileValidationError(err, ErrCoverUnavailable)
			}
		}
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
		return PublishPostResult{}, err
	}
	post.PullEvents()
	if err := post.Publish(domain.NewPostPublishPolicy(defaultMinPublishedPlainTextRunes), domain.PublishInput{
		DraftBody:   post.DraftBody(),
		PublishedAt: s.clock.Now(),
	}); err != nil {
		return PublishPostResult{}, err
	}
	domainEvents := post.PullEvents()

	publishedAt := s.clock.Now()
	snapshot, err := s.bodies.WriteSnapshotBody(ctx, ports.WriteBodyInput{
		PostPublicID:  cmd.PostID,
		OwnerID:       cmd.Actor.UserID,
		SchemaVersion: draftBody.SchemaVersion,
		Blocks:        draftBody.Blocks,
		CanonicalJSON: normalized.CanonicalJSON,
		PlainText:     normalized.PlainText,
		ContentHash:   normalized.ContentHash,
		SizeBytes:     normalized.SizeBytes,
		BlockCount:    normalized.BlockCount,
		CreatedAt:     publishedAt,
	})
	if err != nil {
		return PublishPostResult{}, fmt.Errorf("%w: write published snapshot", ErrDependencyUnavailable)
	}

	var published ports.PostRecord
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		var err error
		published, err = s.posts.Publish(ctx, tx, ports.PublishPostUpdate{
			PublicID:                 cmd.PostID,
			OwnerID:                  cmd.Actor.UserID,
			BasePostVersion:          cmd.BasePostVersion,
			ExpectedDraftBodyID:      cmd.DraftBodyID,
			ExpectedDraftBodyHash:    cmd.DraftBodyHash,
			NewPublishedBodyID:       snapshot.ID,
			NewPublishedBodyHash:     normalized.ContentHash,
			NewPublishedPlainTextLen: len([]rune(normalized.PlainText)),
			PublishedAt:              publishedAt,
		})
		if err != nil {
			return err
		}
		if s.outbox != nil && hasPostPublishedEvent(domainEvents) {
			event, err := newPostPublishedOutboxEvent(current, published, snapshot.ID, normalized.ContentHash, publishedAt)
			if err != nil {
				return err
			}
			if err := s.outbox.Append(ctx, tx, event); err != nil {
				return err
			}
		}
		if current.DraftBodyID != "" && s.cleanup != nil {
			return s.cleanup.Append(ctx, tx, ports.BodyCleanupTask{
				PostID:    current.ID,
				BodyID:    current.DraftBodyID,
				TaskType:  "OLD_DRAFT",
				Reason:    "post_published",
				CreatedAt: publishedAt,
			})
		}
		return nil
	})
	if err != nil {
		// Snapshot candidates become live only after the PostgreSQL transaction
		// switches published_body_id. Commit outcome can be ambiguous, so the
		// application does not directly delete MongoDB bodies here; cleanup
		// workers must confirm PostgreSQL no longer references the body first.
		var cleanupErr error
		if s.cleanup != nil {
			cleanupErr = s.cleanup.AppendOutsideTx(ctx, ports.BodyCleanupTask{
				PostID:    current.ID,
				BodyID:    snapshot.ID,
				TaskType:  "ORPHAN_SNAPSHOT",
				Reason:    "publish_tx_failed",
				CreatedAt: publishedAt,
			})
		}
		if cleanupErr != nil {
			return PublishPostResult{}, fmt.Errorf("%w: publish post; register orphan snapshot cleanup: %w", ErrDependencyUnavailable, cleanupErr)
		}
		if errors.Is(err, domain.ErrDraftConflict) || errors.Is(err, domain.ErrForbidden) ||
			errors.Is(err, domain.ErrPostDeleted) || errors.Is(err, domain.ErrPostAlreadyPublished) {
			return PublishPostResult{}, err
		}
		return PublishPostResult{}, fmt.Errorf("%w: publish post", ErrDependencyUnavailable)
	}

	return PublishPostResult{
		PostID:      published.PublicID,
		PostVersion: published.PostVersion,
		PublishedAt: publishedAt,
	}, nil
}

const defaultMinPublishedPlainTextRunes = 10
