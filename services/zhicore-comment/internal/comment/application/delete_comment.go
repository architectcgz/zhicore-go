package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func (s *Service) DeleteComment(ctx context.Context, cmd DeleteCommentCommand) (DeleteCommentResult, error) {
	return s.deleteComment(ctx, deleteCommentInput{
		actorUserID:   cmd.ActorUserID,
		postID:        cmd.PostID,
		commentID:     cmd.CommentID,
		deletedByRole: DeletedByRoleAuthor,
	})
}

func (s *Service) AdminDeleteComment(ctx context.Context, cmd AdminDeleteCommentCommand) (DeleteCommentResult, error) {
	return s.deleteComment(ctx, deleteCommentInput{
		actorUserID:   cmd.ActorUserID,
		postID:        cmd.PostID,
		commentID:     cmd.CommentID,
		deletedByRole: DeletedByRoleAdmin,
		deleteReason:  strings.TrimSpace(cmd.Reason),
		allowDeleted:  true,
		requireReason: true,
	})
}

type deleteCommentInput struct {
	actorUserID   UserID
	postID        PostID
	commentID     PublicCommentID
	deletedByRole DeletedByRole
	deleteReason  string
	allowDeleted  bool
	requireReason bool
}

func (s *Service) deleteComment(ctx context.Context, input deleteCommentInput) (DeleteCommentResult, error) {
	now := s.clock.Now()
	actorID := domain.UserID(input.actorUserID)
	postID := domain.PostID(strings.TrimSpace(string(input.postID)))
	commentID, err := s.ids.Decode(domain.PublicCommentID(strings.TrimSpace(string(input.commentID))))
	if actorID <= 0 || postID == "" || err != nil {
		return DeleteCommentResult{}, ErrInvalidRequest
	}
	if input.requireReason && strings.TrimSpace(input.deleteReason) == "" {
		return DeleteCommentResult{}, ErrInvalidRequest
	}

	var deleted ports.DeleteSubtreeResult
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		entry, err := s.commands.FindCommentForMutation(txCtx, postID, commentID)
		if err != nil {
			return mapCommentLookupError(err)
		}
		if !input.allowDeleted && entry.Status != domain.CommentStatusNormal {
			return ErrCommentNotFound
		}
		if input.deletedByRole == DeletedByRoleAuthor && entry.AuthorID != actorID {
			return ErrForbidden
		}

		deleted, err = s.commands.SoftDeleteSubtree(txCtx, ports.DeleteSubtreeInput{
			PostID:        postID,
			CommentID:     commentID,
			DeletedBy:     actorID,
			DeletedByRole: string(input.deletedByRole),
			DeleteReason:  input.deleteReason,
			DeletedAt:     now,
			AllowDeleted:  input.allowDeleted,
		})
		if err != nil {
			return mapCommentLookupError(err)
		}
		if deleted.AffectedCount == 0 {
			return nil
		}
		if !deleted.Entry.IsTopLevel() {
			if err := s.stats.DecrementReplyCount(txCtx, deleted.RootID, deleted.AffectedCount, now); err != nil {
				return fmt.Errorf("decrement root reply count: %w", err)
			}
		}
		if err := s.postStats.DecrementForDelete(txCtx, postID, deleted.AffectedCount, deleted.Entry.IsTopLevel(), now); err != nil {
			return fmt.Errorf("decrement post stats: %w", err)
		}
		if deleted.Entry.IsTopLevel() {
			if err := s.commands.HideTopLevelRanks(txCtx, deleted.Entry.ID, now); err != nil {
				return fmt.Errorf("hide top level ranks: %w", err)
			}
		}
		return s.publishDeleted(txCtx, deleted, actorID, input.deletedByRole, input.deleteReason, now)
	}); err != nil {
		return DeleteCommentResult{}, err
	}

	return DeleteCommentResult{
		PostID:         PostID(postID),
		CommentID:      PublicCommentID(s.ids.Encode(commentID)),
		RootCommentID:  rootPublicID(s.ids, deleted.Entry),
		DeletedAt:      now,
		DeletedByRole:  input.deletedByRole,
		AffectedCount:  deleted.AffectedCount,
		AlreadyDeleted: deleted.AlreadyDeleted,
	}, nil
}
