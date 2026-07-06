package application

import (
	"context"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func (s *Service) ensureMediaReferences(ctx context.Context, input domain.CommentMediaInput) error {
	return s.files.EnsureCommentMediaReferenced(ctx, ports.CommentMediaReferences{
		ImageFileIDs:  input.ImageFileIDs,
		VoiceFileID:   input.VoiceFileID,
		VoiceDuration: input.VoiceDuration,
	})
}

func (s *Service) ensureCommentAllowedByRelations(ctx context.Context, actorID domain.UserID, blockers ...domain.UserID) error {
	pairs := make([]ports.BlockPair, 0, len(blockers))
	seen := map[domain.UserID]bool{}
	for _, blockerID := range blockers {
		if blockerID == 0 || blockerID == actorID || seen[blockerID] {
			continue
		}
		seen[blockerID] = true
		pairs = append(pairs, ports.BlockPair{BlockerID: blockerID, BlockedID: actorID})
	}
	if len(pairs) == 0 {
		return nil
	}
	blocked, err := s.userRelations.BatchCheckBlocked(ctx, pairs)
	if err != nil {
		return mapGuardError(err)
	}
	for _, pair := range pairs {
		if blocked[pair] {
			return ErrInteractionBlocked
		}
	}
	return nil
}
