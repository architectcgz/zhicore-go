package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

const (
	commentCreatedEventType = "comment.created"
	commentDeletedEventType = "comment.deleted"
	commentLikedEventType   = "comment.liked"
	commentUnlikedEventType = "comment.unliked"
)

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrDependencyUnavailable  = ports.ErrDependencyUnavailable
	ErrPostNotFound           = ports.ErrPostNotFound
	ErrCommentNotFound        = domain.ErrCommentNotFound
	ErrForbidden              = errors.New("forbidden")
	ErrInteractionBlocked     = ports.ErrInteractionBlocked
	ErrCommentContentRequired = domain.ErrCommentContentRequired
	ErrCommentContentTooLong  = domain.ErrCommentContentTooLong
	ErrParentCommentNotFound  = domain.ErrParentCommentNotFound
	ErrRootCommentNotFound    = domain.ErrRootCommentNotFound
	ErrCommentIDInvalid       = domain.ErrCommentIDInvalid
)

type UserID int64
type PostID string
type PublicCommentID string
type CommentStatus string
type CommentSort string
type DeletedByRole string

const (
	CommentStatusNormal  CommentStatus = "NORMAL"
	CommentStatusDeleted CommentStatus = "DELETED"

	CommentSortRecommended CommentSort = "RECOMMENDED"
	CommentSortHot         CommentSort = "HOT"
	CommentSortTime        CommentSort = "TIME"

	DeletedByRoleAuthor DeletedByRole = "AUTHOR"
	DeletedByRoleAdmin  DeletedByRole = "ADMIN"
)

type Dependencies struct {
	Commands      ports.CommentCommandRepository
	Queries       ports.CommentQueryRepository
	Stats         ports.CommentStatsRepository
	PostStats     ports.CommentPostStatsRepository
	ContentPosts  ports.ContentPostClient
	UserProfiles  ports.UserProfileClient
	UserRelations ports.UserRelationClient
	Files         ports.FileReferenceClient
	IDs           ports.CommentIDCodec
	RateLimiter   ports.RateLimiter
	TxRunner      ports.TransactionRunner
	Outbox        ports.OutboxPublisher
	Clock         ports.Clock
}

type Service struct {
	commands      ports.CommentCommandRepository
	queries       ports.CommentQueryRepository
	stats         ports.CommentStatsRepository
	postStats     ports.CommentPostStatsRepository
	contentPosts  ports.ContentPostClient
	userProfiles  ports.UserProfileClient
	userRelations ports.UserRelationClient
	files         ports.FileReferenceClient
	ids           ports.CommentIDCodec
	rateLimiter   ports.RateLimiter
	txRunner      ports.TransactionRunner
	outbox        ports.OutboxPublisher
	clock         ports.Clock
}

type CreateCommentCommand struct {
	ActorUserID     UserID
	PostID          PostID
	ParentCommentID PublicCommentID
	Content         string
	ImageFileIDs    []string
	VoiceFileID     string
	VoiceDuration   int
}

type CreateCommentResult struct {
	PostID          PostID
	CommentID       PublicCommentID
	RootCommentID   PublicCommentID
	ParentCommentID PublicCommentID
	CreatedAt       time.Time
}

type ListTopLevelCommentsQuery struct {
	PostID       PostID
	ViewerUserID UserID
	Page         int
	Size         int
	Sort         CommentSort
}

type GetCommentDetailQuery struct {
	PostID       PostID
	CommentID    PublicCommentID
	ViewerUserID UserID
}

type ListRepliesByPageQuery struct {
	PostID        PostID
	RootCommentID PublicCommentID
	ViewerUserID  UserID
	Page          int
	Size          int
	Sort          CommentSort
}

type DeleteCommentCommand struct {
	ActorUserID UserID
	PostID      PostID
	CommentID   PublicCommentID
}

type AdminDeleteCommentCommand struct {
	ActorUserID UserID
	PostID      PostID
	CommentID   PublicCommentID
	Reason      string
}

type DeleteCommentResult struct {
	PostID         PostID
	CommentID      PublicCommentID
	RootCommentID  PublicCommentID
	DeletedAt      time.Time
	DeletedByRole  DeletedByRole
	AffectedCount  int
	AlreadyDeleted bool
}

type LikeCommentCommand struct {
	ActorUserID UserID
	PostID      PostID
	CommentID   PublicCommentID
}

type UnlikeCommentCommand = LikeCommentCommand

type LikeCommentResult struct {
	PostID     PostID
	CommentID  PublicCommentID
	Liked      bool
	Changed    bool
	OccurredAt time.Time
}

type GetLikeStatusQuery struct {
	PostID       PostID
	CommentID    PublicCommentID
	ViewerUserID UserID
}

type LikeStatusResult struct {
	PostID    PostID
	CommentID PublicCommentID
	Liked     bool
}

type TopLevelCommentPage struct {
	Items                 []CommentItem
	Page                  int
	Size                  int
	TotalComments         int64
	TotalTopLevelComments int64
	Pages                 int
}

type CommentPage struct {
	Items []CommentItem
	Page  int
	Size  int
	Total int64
	Pages int
}

type CommentItem struct {
	PostID          PostID
	CommentID       PublicCommentID
	RootCommentID   PublicCommentID
	ParentCommentID PublicCommentID
	Author          AuthorSummary
	Content         string
	ImageFileIDs    []string
	VoiceFileID     string
	VoiceDuration   int
	Status          CommentStatus
	Stats           CommentStats
	Viewer          *ViewerState
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type AuthorSummary struct {
	PublicID     string
	DisplayName  string
	AvatarFileID string
	AvatarURL    string
	Unavailable  bool
}

type CommentStats struct {
	LikeCount  int64
	ReplyCount int64
}

type ViewerState struct {
	Liked bool
}

func NewService(deps Dependencies) (*Service, error) {
	for _, item := range []struct {
		name  string
		value any
	}{
		{"Commands", deps.Commands},
		{"Queries", deps.Queries},
		{"Stats", deps.Stats},
		{"PostStats", deps.PostStats},
		{"ContentPosts", deps.ContentPosts},
		{"UserProfiles", deps.UserProfiles},
		{"UserRelations", deps.UserRelations},
		{"Files", deps.Files},
		{"IDs", deps.IDs},
		{"RateLimiter", deps.RateLimiter},
		{"TxRunner", deps.TxRunner},
		{"Outbox", deps.Outbox},
		{"Clock", deps.Clock},
	} {
		if item.value == nil {
			return nil, fmt.Errorf("%s is required", item.name)
		}
	}
	return &Service{
		commands:      deps.Commands,
		queries:       deps.Queries,
		stats:         deps.Stats,
		postStats:     deps.PostStats,
		contentPosts:  deps.ContentPosts,
		userProfiles:  deps.UserProfiles,
		userRelations: deps.UserRelations,
		files:         deps.Files,
		ids:           deps.IDs,
		rateLimiter:   deps.RateLimiter,
		txRunner:      deps.TxRunner,
		outbox:        deps.Outbox,
		clock:         deps.Clock,
	}, nil
}

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

func (s *Service) ListTopLevelCommentsByPage(ctx context.Context, query ListTopLevelCommentsQuery) (TopLevelCommentPage, error) {
	normalized, err := normalizeTopLevelPageQuery(query)
	if err != nil {
		return TopLevelCommentPage{}, err
	}
	postID := domain.PostID(strings.TrimSpace(string(normalized.PostID)))
	if _, err := s.contentPosts.CheckPostCommentable(ctx, postID); err != nil {
		return TopLevelCommentPage{}, mapGuardError(err)
	}
	postStats, err := s.postStats.Get(ctx, postID)
	if err != nil {
		return TopLevelCommentPage{}, mapGuardError(err)
	}
	sort := domainCommentSort(normalized.Sort)
	records, err := s.queries.ListTopLevelComments(ctx, ports.TopLevelCommentPageQuery{
		PostID: postID,
		Page:   normalized.Page,
		Size:   normalized.Size,
		Sort:   sort,
	})
	if err != nil {
		return TopLevelCommentPage{}, mapGuardError(err)
	}

	authorSummaries := s.loadAuthorSummaries(ctx, records.Items)
	viewerLiked, err := s.loadViewerLiked(ctx, domain.UserID(normalized.ViewerUserID), records.Items)
	if err != nil {
		return TopLevelCommentPage{}, mapGuardError(err)
	}

	items := make([]CommentItem, 0, len(records.Items))
	for _, record := range records.Items {
		items = append(items, s.commentItem(record, authorSummaries[record.Comment.AuthorID], viewerLiked))
	}
	return TopLevelCommentPage{
		Items:                 items,
		Page:                  normalized.Page,
		Size:                  normalized.Size,
		TotalComments:         postStats.TotalComments,
		TotalTopLevelComments: postStats.TotalTopLevelComments,
		Pages:                 pageCount(postStats.TotalTopLevelComments, normalized.Size),
	}, nil
}

func (s *Service) GetCommentDetail(ctx context.Context, query GetCommentDetailQuery) (CommentItem, error) {
	postID := domain.PostID(strings.TrimSpace(string(query.PostID)))
	commentID, err := s.ids.Decode(domain.PublicCommentID(strings.TrimSpace(string(query.CommentID))))
	if postID == "" || err != nil {
		return CommentItem{}, ErrInvalidRequest
	}
	if _, err := s.contentPosts.CheckPostCommentable(ctx, postID); err != nil {
		return CommentItem{}, mapGuardError(err)
	}
	record, err := s.queries.GetCommentDetail(ctx, postID, commentID)
	if err != nil {
		return CommentItem{}, mapCommentLookupError(err)
	}

	authorSummaries := s.loadAuthorSummaries(ctx, []ports.TopLevelCommentRecord{record})
	viewerLiked, err := s.loadViewerLiked(ctx, domain.UserID(query.ViewerUserID), []ports.TopLevelCommentRecord{record})
	if err != nil {
		return CommentItem{}, mapGuardError(err)
	}
	return s.commentItem(record, authorSummaries[record.Comment.AuthorID], viewerLiked), nil
}

func (s *Service) ListRepliesByPage(ctx context.Context, query ListRepliesByPageQuery) (CommentPage, error) {
	normalized, err := normalizeRepliesPageQuery(query)
	if err != nil {
		return CommentPage{}, err
	}
	postID := domain.PostID(strings.TrimSpace(string(normalized.PostID)))
	rootID, err := s.ids.Decode(domain.PublicCommentID(strings.TrimSpace(string(normalized.RootCommentID))))
	if err != nil {
		return CommentPage{}, ErrInvalidRequest
	}
	if _, err := s.contentPosts.CheckPostCommentable(ctx, postID); err != nil {
		return CommentPage{}, mapGuardError(err)
	}
	records, err := s.queries.ListRepliesByPage(ctx, ports.ReplyCommentPageQuery{
		PostID: postID,
		RootID: rootID,
		Page:   normalized.Page,
		Size:   normalized.Size,
		Sort:   domainCommentSort(normalized.Sort),
	})
	if err != nil {
		return CommentPage{}, mapDomainValidationError(err)
	}

	authorSummaries := s.loadAuthorSummaries(ctx, records.Items)
	viewerLiked, err := s.loadViewerLiked(ctx, domain.UserID(normalized.ViewerUserID), records.Items)
	if err != nil {
		return CommentPage{}, mapGuardError(err)
	}
	items := make([]CommentItem, 0, len(records.Items))
	for _, record := range records.Items {
		items = append(items, s.commentItem(record, authorSummaries[record.Comment.AuthorID], viewerLiked))
	}
	return CommentPage{
		Items: items,
		Page:  normalized.Page,
		Size:  normalized.Size,
		Total: records.Total,
		Pages: pageCount(records.Total, normalized.Size),
	}, nil
}

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

func normalizeTopLevelPageQuery(query ListTopLevelCommentsQuery) (ListTopLevelCommentsQuery, error) {
	query.PostID = PostID(strings.TrimSpace(string(query.PostID)))
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Size == 0 {
		query.Size = 20
	}
	if query.Sort == "" {
		query.Sort = CommentSortRecommended
	}
	if query.PostID == "" || query.Page < 1 || query.Size < 1 || query.Size > 100 {
		return ListTopLevelCommentsQuery{}, ErrInvalidRequest
	}
	switch query.Sort {
	case CommentSortRecommended, CommentSortHot, CommentSortTime:
	default:
		return ListTopLevelCommentsQuery{}, ErrInvalidRequest
	}
	return query, nil
}

func normalizeRepliesPageQuery(query ListRepliesByPageQuery) (ListRepliesByPageQuery, error) {
	query.PostID = PostID(strings.TrimSpace(string(query.PostID)))
	query.RootCommentID = PublicCommentID(strings.TrimSpace(string(query.RootCommentID)))
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Size == 0 {
		query.Size = 20
	}
	if query.Sort == "" {
		query.Sort = CommentSortHot
	}
	if query.PostID == "" || query.RootCommentID == "" || query.Page < 1 || query.Size < 1 || query.Size > 100 {
		return ListRepliesByPageQuery{}, ErrInvalidRequest
	}
	switch query.Sort {
	case CommentSortHot, CommentSortTime:
	default:
		return ListRepliesByPageQuery{}, ErrInvalidRequest
	}
	return query, nil
}

func domainCommentSort(sort CommentSort) domain.CommentSort {
	switch sort {
	case CommentSortHot:
		return domain.CommentSortHot
	case CommentSortTime:
		return domain.CommentSortTime
	default:
		return domain.CommentSortRecommended
	}
}

func (s *Service) loadAuthorSummaries(ctx context.Context, records []ports.TopLevelCommentRecord) map[domain.UserID]ports.AuthorSummary {
	userIDs := make([]domain.UserID, 0, len(records))
	seen := map[domain.UserID]bool{}
	for _, record := range records {
		if record.Comment.AuthorID == 0 || seen[record.Comment.AuthorID] {
			continue
		}
		seen[record.Comment.AuthorID] = true
		userIDs = append(userIDs, record.Comment.AuthorID)
	}
	if len(userIDs) == 0 {
		return map[domain.UserID]ports.AuthorSummary{}
	}
	summaries, err := s.userProfiles.BatchGetAuthorSummaries(ctx, userIDs)
	if err == nil {
		return summaries
	}
	degraded := make(map[domain.UserID]ports.AuthorSummary, len(userIDs))
	for _, userID := range userIDs {
		degraded[userID] = ports.AuthorSummary{UserID: userID, Unavailable: true}
	}
	return degraded
}

func (s *Service) loadViewerLiked(ctx context.Context, viewerID domain.UserID, records []ports.TopLevelCommentRecord) (map[domain.CommentID]bool, error) {
	if viewerID <= 0 || len(records) == 0 {
		return map[domain.CommentID]bool{}, nil
	}
	ids := make([]domain.CommentID, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.Comment.ID)
	}
	return s.queries.BatchGetViewerLiked(ctx, viewerID, ids)
}

func (s *Service) commentItem(record ports.TopLevelCommentRecord, author ports.AuthorSummary, viewerLiked map[domain.CommentID]bool) CommentItem {
	comment := record.Comment
	item := CommentItem{
		PostID:        PostID(comment.PostID),
		CommentID:     PublicCommentID(s.ids.Encode(comment.ID)),
		Author:        AuthorSummary{PublicID: author.PublicID, DisplayName: author.DisplayName, AvatarFileID: author.AvatarFileID, AvatarURL: author.AvatarURL, Unavailable: author.Unavailable},
		Content:       comment.Content,
		ImageFileIDs:  append([]string(nil), comment.Media.ImageFileIDs...),
		VoiceFileID:   comment.Media.VoiceFileID,
		VoiceDuration: comment.Media.VoiceDuration,
		Status:        CommentStatus(comment.Status),
		Stats:         CommentStats{LikeCount: record.Stats.LikeCount, ReplyCount: record.Stats.ReplyCount},
		CreatedAt:     comment.CreatedAt,
		UpdatedAt:     comment.UpdatedAt,
	}
	if comment.IsReply() {
		item.RootCommentID = PublicCommentID(s.ids.Encode(comment.RootID))
		item.ParentCommentID = PublicCommentID(s.ids.Encode(comment.ParentID))
	}
	if viewerLiked != nil {
		if liked, ok := viewerLiked[comment.ID]; ok {
			item.Viewer = &ViewerState{Liked: liked}
		}
	}
	return item
}

func pageCount(total int64, size int) int {
	if total <= 0 || size <= 0 {
		return 0
	}
	return int((total + int64(size) - 1) / int64(size))
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

func (s *Service) publishCreated(ctx context.Context, event domain.CommentCreated, post ports.CommentablePost, occurredAt time.Time) error {
	comment := event.CreatedComment()
	payload := map[string]any{
		"commentId":    comment.ID,
		"publicId":     post.PostID,
		"internalId":   post.ContentInternalID,
		"postAuthorId": post.AuthorID,
		"authorId":     comment.AuthorID,
		"hasImages":    len(comment.Media.ImageFileIDs) > 0,
		"hasVoice":     strings.TrimSpace(comment.Media.VoiceFileID) != "",
		"createdAt":    occurredAt.UTC().Format(time.RFC3339),
	}
	if root, ok := event.RootComment(); ok {
		parent, _ := event.ParentComment()
		payload["rootId"] = root.ID
		payload["rootAuthorId"] = root.AuthorID
		payload["parentId"] = parent.ID
		payload["parentAuthorId"] = parent.AuthorID
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal comment created event: %w", err)
	}
	if err := s.outbox.Publish(ctx, ports.OutboxMessage{
		EventType:     commentCreatedEventType,
		AggregateType: "comment",
		AggregateID:   strconv.FormatInt(int64(comment.ID), 10),
		OccurredAt:    occurredAt,
		Payload:       body,
	}); err != nil {
		return fmt.Errorf("publish comment created outbox: %w", err)
	}
	return nil
}

func (s *Service) publishDeleted(ctx context.Context, deleted ports.DeleteSubtreeResult, deletedBy domain.UserID, role DeletedByRole, reason string, occurredAt time.Time) error {
	entry := deleted.Entry
	payload := map[string]any{
		"commentId":     entry.ID,
		"publicId":      entry.PostID,
		"internalId":    entry.ContentInternalID,
		"authorId":      entry.AuthorID,
		"deletedBy":     deletedBy,
		"deletedByRole": string(role),
		"deletedAt":     occurredAt.UTC().Format(time.RFC3339),
		"isRoot":        entry.IsTopLevel(),
		"affectedCount": deleted.AffectedCount,
	}
	if entry.IsReply() {
		payload["rootId"] = entry.RootID
	}
	if strings.TrimSpace(reason) != "" {
		payload["deleteReason"] = strings.TrimSpace(reason)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal comment deleted event: %w", err)
	}
	return s.outbox.Publish(ctx, ports.OutboxMessage{
		EventType:     commentDeletedEventType,
		AggregateType: "comment",
		AggregateID:   strconv.FormatInt(int64(entry.ID), 10),
		OccurredAt:    occurredAt,
		Payload:       body,
	})
}

func (s *Service) publishLikeChanged(ctx context.Context, eventType string, comment domain.Comment, actorID domain.UserID, occurredAt time.Time) error {
	payload := map[string]any{
		"commentId":       comment.ID,
		"publicId":        comment.PostID,
		"internalId":      comment.ContentInternalID,
		"commentAuthorId": comment.AuthorID,
		"occurredAt":      occurredAt.UTC().Format(time.RFC3339),
	}
	if eventType == commentLikedEventType {
		payload["likedBy"] = actorID
	} else {
		payload["unlikedBy"] = actorID
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s event: %w", eventType, err)
	}
	return s.outbox.Publish(ctx, ports.OutboxMessage{
		EventType:     eventType,
		AggregateType: "comment",
		AggregateID:   strconv.FormatInt(int64(comment.ID), 10),
		OccurredAt:    occurredAt,
		Payload:       body,
	})
}

func rootPublicID(ids ports.CommentIDCodec, comment domain.Comment) PublicCommentID {
	if !comment.IsReply() {
		return ""
	}
	return PublicCommentID(ids.Encode(comment.RootID))
}

func mapCommentLookupError(err error) error {
	if errors.Is(err, domain.ErrCommentNotFound) || errors.Is(err, domain.ErrParentCommentNotFound) || errors.Is(err, domain.ErrRootCommentNotFound) {
		return ErrCommentNotFound
	}
	return mapDomainValidationError(err)
}

func mapDomainValidationError(err error) error {
	switch {
	case errors.Is(err, domain.ErrCommentContentRequired):
		return ErrCommentContentRequired
	case errors.Is(err, domain.ErrCommentContentTooLong):
		return ErrCommentContentTooLong
	case errors.Is(err, domain.ErrCommentMediaInvalid), errors.Is(err, domain.ErrPostIDInvalid), errors.Is(err, domain.ErrUserIDInvalid):
		return ErrInvalidRequest
	case errors.Is(err, domain.ErrParentCommentNotFound), errors.Is(err, domain.ErrCommentNotFound):
		return ErrParentCommentNotFound
	case errors.Is(err, domain.ErrRootCommentNotFound):
		return ErrRootCommentNotFound
	case errors.Is(err, domain.ErrCommentIDInvalid):
		return ErrCommentIDInvalid
	default:
		return err
	}
}

func mapGuardError(err error) error {
	switch {
	case errors.Is(err, ports.ErrDependencyUnavailable):
		return ErrDependencyUnavailable
	case errors.Is(err, ports.ErrPostNotFound):
		return ErrPostNotFound
	case errors.Is(err, ports.ErrUserUnavailable):
		return ErrForbidden
	case errors.Is(err, ports.ErrInteractionBlocked):
		return ErrInteractionBlocked
	default:
		return err
	}
}
