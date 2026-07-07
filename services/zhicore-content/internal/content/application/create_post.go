package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Service) CreatePost(ctx context.Context, cmd CreatePostCommand) (CreatePostResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return CreatePostResult{}, ErrLoginRequired
	}

	title, err := domain.NewPostTitle(cmd.Title)
	if err != nil {
		return CreatePostResult{}, err
	}
	if err := s.enforceRateLimit(ctx, actorRateLimitRequest(ports.RateLimitTypeDraftWrite, cmd.Actor, "post", "create_post")); err != nil {
		return CreatePostResult{}, err
	}

	owner, err := s.users.GetOwnerSnapshot(ctx, cmd.Actor.UserID)
	if err != nil {
		return CreatePostResult{}, fmt.Errorf("%w: get owner snapshot", ErrDependencyUnavailable)
	}

	var draftBody ports.StoredBody
	var normalized ports.NormalizedBody
	if cmd.Body != nil {
		normalized, err = s.parser.Parse(ctx, ports.PostBodyWriteInput{
			SchemaVersion: cmd.Body.SchemaVersion,
			Blocks:        cmd.Body.Blocks,
		})
		if err != nil {
			return CreatePostResult{}, err
		}
		if s.files != nil && len(normalized.MediaRefs) > 0 {
			if err := s.files.ValidateBodyMediaRefs(ctx, normalized.MediaRefs); err != nil {
				return CreatePostResult{}, mapFileValidationError(err, ErrMediaRefInvalid)
			}
		}

		// Body writes are copy-on-write candidates. The PostgreSQL row is only
		// created after MongoDB accepts the draft body, so a body-store failure
		// cannot create a visible post that points at missing content.
		draftBody, err = s.bodies.WriteDraftBody(ctx, ports.WriteBodyInput{
			OwnerID:       cmd.Actor.UserID,
			SchemaVersion: cmd.Body.SchemaVersion,
			Blocks:        cmd.Body.Blocks,
			CanonicalJSON: normalized.CanonicalJSON,
			PlainText:     normalized.PlainText,
			ContentHash:   normalized.ContentHash,
			SizeBytes:     normalized.SizeBytes,
			BlockCount:    normalized.BlockCount,
			CreatedAt:     s.clock.Now(),
		})
		if err != nil {
			return CreatePostResult{}, fmt.Errorf("%w: write draft body", ErrDependencyUnavailable)
		}
	}

	var created ports.PostRecord
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		input := ports.CreateDraftPost{
			OwnerID:              cmd.Actor.UserID,
			OwnerDisplayName:     owner.DisplayName,
			OwnerAvatarFileID:    owner.AvatarFileID,
			OwnerProfileVersion:  owner.ProfileVersion,
			Title:                string(title),
			Summary:              cmd.Summary,
			CoverFileID:          cmd.CoverFileID,
			TopicID:              cmd.TopicID,
			CategoryID:           cmd.CategoryID,
			Tags:                 append([]string(nil), cmd.Tags...),
			DraftBodyID:          draftBody.ID,
			DraftBodyHash:        normalized.ContentHash,
			DraftSizeBytes:       normalized.SizeBytes,
			DraftPlainTextLength: len([]rune(normalized.PlainText)),
		}
		var err error
		created, err = s.posts.CreateDraft(ctx, tx, input)
		return err
	})
	if err != nil {
		if draftBody.ID != "" && s.cleanup != nil {
			_ = s.cleanup.AppendOutsideTx(ctx, ports.BodyCleanupTask{
				BodyID:    draftBody.ID,
				TaskType:  "ORPHAN_DRAFT",
				Reason:    "draft_replaced",
				CreatedAt: s.clock.Now(),
			})
		}
		if errors.Is(err, ErrTaxonomyReferenceNotFound) {
			return CreatePostResult{}, err
		}
		return CreatePostResult{}, fmt.Errorf("%w: create draft", ErrDependencyUnavailable)
	}

	return CreatePostResult{
		PostID:      created.PublicID,
		PostVersion: created.PostVersion,
	}, nil
}
