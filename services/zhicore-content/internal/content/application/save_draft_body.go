package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Service) SaveDraftBody(ctx context.Context, cmd SaveDraftBodyCommand) (SaveDraftBodyResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return SaveDraftBodyResult{}, ErrLoginRequired
	}
	if err := s.enforceRateLimit(ctx, actorRateLimitRequest(ports.RateLimitTypeDraftWrite, cmd.Actor, cmd.PostID, "save_draft_body")); err != nil {
		return SaveDraftBodyResult{}, err
	}

	current, err := s.loadPostForDraftWrite(ctx, cmd.PostID)
	if err != nil {
		return SaveDraftBodyResult{}, err
	}
	if current.OwnerID != cmd.Actor.UserID {
		return SaveDraftBodyResult{}, domain.ErrForbidden
	}
	if current.Status == domain.PostStatusDeleted {
		return SaveDraftBodyResult{}, domain.ErrPostDeleted
	}
	if current.Status == domain.PostStatusScheduled {
		// A scheduled post has already captured the exact draft body/hash to
		// publish; require canceling the schedule before edits can move it.
		return SaveDraftBodyResult{}, domain.ErrDraftConflict
	}
	if current.PostVersion != cmd.BasePostVersion ||
		current.DraftBodyID != cmd.BaseDraftBodyID ||
		current.DraftBodyHash != cmd.BaseDraftBodyHash {
		return SaveDraftBodyResult{}, domain.ErrDraftConflict
	}

	normalized, err := s.parser.Parse(ctx, ports.PostBodyWriteInput{
		SchemaVersion: cmd.Body.SchemaVersion,
		Blocks:        cmd.Body.Blocks,
	})
	if err != nil {
		return SaveDraftBodyResult{}, err
	}
	if s.files != nil && len(normalized.MediaRefs) > 0 {
		if err := s.files.ValidateBodyMediaRefs(ctx, normalized.MediaRefs); err != nil {
			return SaveDraftBodyResult{}, mapFileValidationError(err, ErrMediaRefInvalid)
		}
	}

	now := s.clock.Now()
	wordCount := len([]rune(normalized.PlainText))
	if normalized.ContentHash == current.DraftBodyHash {
		return SaveDraftBodyResult{
			PostID:        current.PublicID,
			PostVersion:   current.PostVersion,
			DraftBodyID:   current.DraftBodyID,
			DraftBodyHash: current.DraftBodyHash,
			SavedAt:       now,
			WordCount:     wordCount,
		}, nil
	}

	newDraft, err := s.bodies.WriteDraftBody(ctx, ports.WriteBodyInput{
		PostPublicID:  cmd.PostID,
		OwnerID:       cmd.Actor.UserID,
		SchemaVersion: cmd.Body.SchemaVersion,
		Blocks:        cmd.Body.Blocks,
		CanonicalJSON: normalized.CanonicalJSON,
		PlainText:     normalized.PlainText,
		ContentHash:   normalized.ContentHash,
		SizeBytes:     normalized.SizeBytes,
		BlockCount:    normalized.BlockCount,
		CreatedAt:     now,
	})
	if err != nil {
		return SaveDraftBodyResult{}, fmt.Errorf("%w: write draft body", ErrDependencyUnavailable)
	}

	var saved ports.PostRecord
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		var err error
		saved, err = s.posts.SaveDraftBody(ctx, tx, ports.SaveDraftBodyUpdate{
			PublicID:             cmd.PostID,
			OwnerID:              cmd.Actor.UserID,
			BasePostVersion:      cmd.BasePostVersion,
			BaseDraftBodyID:      cmd.BaseDraftBodyID,
			BaseDraftBodyHash:    cmd.BaseDraftBodyHash,
			NewDraftBodyID:       newDraft.ID,
			NewDraftBodyHash:     normalized.ContentHash,
			NewDraftSizeBytes:    normalized.SizeBytes,
			NewDraftPlainTextLen: wordCount,
		})
		if err != nil {
			return err
		}
		if current.DraftBodyID != "" && current.DraftBodyID != newDraft.ID && s.cleanup != nil {
			return s.cleanup.Append(ctx, tx, ports.BodyCleanupTask{
				PostID:    current.ID,
				BodyID:    current.DraftBodyID,
				TaskType:  "OLD_DRAFT",
				Reason:    "draft_replaced",
				CreatedAt: now,
			})
		}
		return nil
	})
	if err != nil {
		// The new MongoDB body is not referenced if the PostgreSQL conditional
		// update fails; record an orphan cleanup outside the rolled-back tx.
		if s.cleanup != nil {
			_ = s.cleanup.AppendOutsideTx(ctx, ports.BodyCleanupTask{
				PostID:    current.ID,
				BodyID:    newDraft.ID,
				TaskType:  "ORPHAN_DRAFT",
				Reason:    "draft_replaced",
				CreatedAt: now,
			})
		}
		if errors.Is(err, domain.ErrDraftConflict) || errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrPostDeleted) {
			return SaveDraftBodyResult{}, err
		}
		return SaveDraftBodyResult{}, fmt.Errorf("%w: save draft body", ErrDependencyUnavailable)
	}

	return SaveDraftBodyResult{
		PostID:        saved.PublicID,
		PostVersion:   saved.PostVersion,
		DraftBodyID:   saved.DraftBodyID,
		DraftBodyHash: saved.DraftBodyHash,
		SavedAt:       now,
		WordCount:     wordCount,
	}, nil
}

func (s *Service) loadPostForDraftWrite(ctx context.Context, publicID string) (ports.PostRecord, error) {
	var current ports.PostRecord
	err := s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		var err error
		current, err = s.posts.GetForUpdate(ctx, tx, publicID)
		return err
	})
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return ports.PostRecord{}, err
		}
		return ports.PostRecord{}, fmt.Errorf("%w: load post", ErrDependencyUnavailable)
	}
	return current, nil
}
