package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func (s *Service) GetLikeStatus(ctx context.Context, query GetLikeStatusQuery) (LikeStatusResult, error) {
	postID := domain.PostID(strings.TrimSpace(string(query.PostID)))
	commentID, err := s.ids.Decode(domain.PublicCommentID(strings.TrimSpace(string(query.CommentID))))
	viewerID := domain.UserID(query.ViewerUserID)
	if postID == "" || viewerID <= 0 || err != nil {
		return LikeStatusResult{}, ErrInvalidRequest
	}
	if _, err := s.queries.GetCommentDetail(ctx, postID, commentID); err != nil {
		return LikeStatusResult{}, mapCommentLookupError(err)
	}
	liked, err := s.queries.BatchGetViewerLiked(ctx, viewerID, []domain.CommentID{commentID})
	if err != nil {
		return LikeStatusResult{}, mapGuardError(err)
	}
	return LikeStatusResult{PostID: PostID(postID), CommentID: PublicCommentID(s.ids.Encode(commentID)), Liked: liked[commentID]}, nil
}

func (s *Service) LikeComment(ctx context.Context, cmd LikeCommentCommand) (LikeCommentResult, error) {
	return s.changeLike(ctx, cmd, true)
}

func (s *Service) UnlikeComment(ctx context.Context, cmd UnlikeCommentCommand) (LikeCommentResult, error) {
	return s.changeLike(ctx, LikeCommentCommand(cmd), false)
}

func (s *Service) changeLike(ctx context.Context, cmd LikeCommentCommand, liked bool) (LikeCommentResult, error) {
	now := s.clock.Now()
	actorID := domain.UserID(cmd.ActorUserID)
	postID := domain.PostID(strings.TrimSpace(string(cmd.PostID)))
	commentID, err := s.ids.Decode(domain.PublicCommentID(strings.TrimSpace(string(cmd.CommentID))))
	if actorID <= 0 || postID == "" || err != nil {
		return LikeCommentResult{}, ErrInvalidRequest
	}

	var changed bool
	var comment domain.Comment
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		comment, err = s.commands.FindCommentForMutation(txCtx, postID, commentID)
		if err != nil {
			return mapCommentLookupError(err)
		}
		if liked && comment.Status != domain.CommentStatusNormal {
			return ErrCommentNotFound
		}
		if liked {
			if err := s.ensureCommentAllowedByRelations(txCtx, actorID, comment.AuthorID); err != nil {
				return err
			}
			changed, err = s.commands.UpsertLike(txCtx, ports.LikeMutationInput{PostID: postID, CommentID: commentID, UserID: actorID, Now: now})
		} else {
			changed, err = s.commands.DeleteLike(txCtx, ports.LikeMutationInput{PostID: postID, CommentID: commentID, UserID: actorID, Now: now})
		}
		if err != nil {
			return fmt.Errorf("change comment like state: %w", err)
		}
		if !changed {
			return nil
		}
		delta := 1
		eventType := commentLikedEventType
		if !liked {
			delta = -1
			eventType = commentUnlikedEventType
		}
		if err := s.commands.AppendCounterDelta(txCtx, ports.CommentCounterDelta{
			CommentID:   comment.ID,
			PostID:      comment.PostID,
			CounterType: "LIKE",
			DeltaValue:  delta,
			CreatedAt:   now,
		}); err != nil {
			return fmt.Errorf("append comment counter delta: %w", err)
		}
		return s.publishLikeChanged(txCtx, eventType, comment, actorID, now)
	}); err != nil {
		return LikeCommentResult{}, err
	}

	return LikeCommentResult{
		PostID:     PostID(postID),
		CommentID:  PublicCommentID(s.ids.Encode(commentID)),
		Liked:      liked,
		Changed:    changed,
		OccurredAt: now,
	}, nil
}
