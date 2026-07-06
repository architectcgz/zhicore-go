package application

import (
	"context"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

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
