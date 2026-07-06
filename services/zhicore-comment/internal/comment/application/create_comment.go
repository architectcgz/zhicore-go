package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func (s *Service) CreateComment(ctx context.Context, cmd CreateCommentCommand) (CreateCommentResult, error) {
	now := s.clock.Now()
	actorID := domain.UserID(cmd.ActorUserID)
	postID := domain.PostID(strings.TrimSpace(string(cmd.PostID)))
	parentCommentID := domain.PublicCommentID(strings.TrimSpace(string(cmd.ParentCommentID)))
	if actorID <= 0 || strings.TrimSpace(string(postID)) == "" {
		return CreateCommentResult{}, ErrInvalidRequest
	}
	mediaInput := domain.CommentMediaInput{ImageFileIDs: cmd.ImageFileIDs, VoiceFileID: cmd.VoiceFileID, VoiceDuration: cmd.VoiceDuration}
	if _, _, err := domain.NewCommentBody(cmd.Content, mediaInput); err != nil {
		return CreateCommentResult{}, mapDomainValidationError(err)
	}

	post, err := s.contentPosts.CheckPostCommentable(ctx, postID)
	if err != nil {
		return CreateCommentResult{}, mapGuardError(err)
	}
	if err := s.userProfiles.EnsureUserCanComment(ctx, actorID); err != nil {
		return CreateCommentResult{}, mapGuardError(err)
	}
	if err := s.ensureMediaReferences(ctx, mediaInput); err != nil {
		return CreateCommentResult{}, mapGuardError(err)
	}
	if err := s.rateLimiter.AllowCreateComment(ctx, ports.CreateCommentRateLimitInput{ActorUserID: actorID, PostID: postID}); err != nil {
		return CreateCommentResult{}, mapGuardError(err)
	}
	if parentCommentID == "" {
		if err := s.ensureCommentAllowedByRelations(ctx, actorID, post.AuthorID); err != nil {
			return CreateCommentResult{}, err
		}
	} else {
		// 回复写入的拉黑 guard 属于外部 User 事实，不能放进本地写事务。
		// 事务外预读只用于拿 parentAuthorId；父评论存在性、状态和树结构仍由事务内 authoritative read 决定。
		parentID, err := s.ids.Decode(parentCommentID)
		if err != nil {
			return CreateCommentResult{}, ErrCommentIDInvalid
		}
		preview, ok, err := s.commands.FindReplyGuardPreview(ctx, postID, parentID)
		if err != nil {
			return CreateCommentResult{}, mapGuardError(err)
		}
		if ok {
			if err := s.ensureCommentAllowedByRelations(ctx, actorID, post.AuthorID, preview.ParentAuthorID); err != nil {
				return CreateCommentResult{}, err
			}
		}
	}

	var created domain.Comment
	var createdEvent domain.CommentCreated
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		if parentCommentID == "" {
			created, createdEvent, err = s.createTopLevel(txCtx, post, actorID, cmd, mediaInput, now)
			return err
		}
		target, err := s.replyTarget(txCtx, postID, parentCommentID)
		if err != nil {
			return err
		}
		created, createdEvent, err = s.createReply(txCtx, post, actorID, cmd, mediaInput, now, target.Root, target.Parent)
		return err
	}); err != nil {
		return CreateCommentResult{}, err
	}

	result := CreateCommentResult{
		PostID:    PostID(created.PostID),
		CommentID: PublicCommentID(s.ids.Encode(created.ID)),
		CreatedAt: created.CreatedAt,
	}
	if root, ok := createdEvent.RootComment(); ok {
		parent, _ := createdEvent.ParentComment()
		result.RootCommentID = PublicCommentID(s.ids.Encode(root.ID))
		result.ParentCommentID = PublicCommentID(s.ids.Encode(parent.ID))
	}
	return result, nil
}

func (s *Service) createTopLevel(ctx context.Context, post ports.CommentablePost, actorID domain.UserID, cmd CreateCommentCommand, mediaInput domain.CommentMediaInput, now time.Time) (domain.Comment, domain.CommentCreated, error) {
	draft, err := domain.NewTopLevelDraft(post.PostID, post.ContentInternalID, actorID, cmd.Content, mediaInput, now)
	if err != nil {
		return domain.Comment{}, nil, mapDomainValidationError(err)
	}
	stored, err := s.commands.Create(ctx, draft)
	if err != nil {
		return domain.Comment{}, nil, fmt.Errorf("create comment: %w", err)
	}
	if err := s.stats.Initialize(ctx, stored.ID, now); err != nil {
		return domain.Comment{}, nil, fmt.Errorf("initialize comment stats: %w", err)
	}
	if err := s.postStats.IncrementForTopLevel(ctx, stored.PostID, now); err != nil {
		return domain.Comment{}, nil, fmt.Errorf("increment post stats: %w", err)
	}
	if err := s.commands.InitializeTopLevelRanks(ctx, stored, now); err != nil {
		return domain.Comment{}, nil, fmt.Errorf("initialize comment ranks: %w", err)
	}
	event, err := domain.NewTopLevelCommentCreated(stored)
	if err != nil {
		return domain.Comment{}, nil, mapDomainValidationError(err)
	}
	if err := s.publishCreated(ctx, event, post, now); err != nil {
		return domain.Comment{}, nil, err
	}
	return stored, event, nil
}

func (s *Service) createReply(ctx context.Context, post ports.CommentablePost, actorID domain.UserID, cmd CreateCommentCommand, mediaInput domain.CommentMediaInput, now time.Time, root, parent domain.Comment) (domain.Comment, domain.CommentCreated, error) {
	draft, err := domain.NewReplyDraft(post.PostID, post.ContentInternalID, actorID, root, parent, cmd.Content, mediaInput, now)
	if err != nil {
		return domain.Comment{}, nil, mapDomainValidationError(err)
	}
	stored, err := s.commands.Create(ctx, draft)
	if err != nil {
		return domain.Comment{}, nil, fmt.Errorf("create reply: %w", err)
	}
	if err := s.stats.Initialize(ctx, stored.ID, now); err != nil {
		return domain.Comment{}, nil, fmt.Errorf("initialize reply stats: %w", err)
	}
	if err := s.stats.IncrementReplyCount(ctx, root.ID, now); err != nil {
		return domain.Comment{}, nil, fmt.Errorf("increment root reply count: %w", err)
	}
	if err := s.postStats.IncrementForReply(ctx, stored.PostID, now); err != nil {
		return domain.Comment{}, nil, fmt.Errorf("increment post stats: %w", err)
	}
	event, err := domain.NewReplyCreated(stored, root, parent)
	if err != nil {
		return domain.Comment{}, nil, mapDomainValidationError(err)
	}
	if err := s.publishCreated(ctx, event, post, now); err != nil {
		return domain.Comment{}, nil, err
	}
	return stored, event, nil
}

func (s *Service) replyTarget(ctx context.Context, postID domain.PostID, publicParentID domain.PublicCommentID) (ports.ReplyTarget, error) {
	parentID, err := s.ids.Decode(publicParentID)
	if err != nil {
		return ports.ReplyTarget{}, ErrCommentIDInvalid
	}
	target, err := s.commands.FindReplyTarget(ctx, postID, parentID)
	if err != nil {
		return ports.ReplyTarget{}, mapDomainValidationError(err)
	}
	return target, nil
}
