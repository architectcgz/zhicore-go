package application

import (
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
)

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

func pageCount(total int64, size int) int {
	if total <= 0 || size <= 0 {
		return 0
	}
	return int((total + int64(size) - 1) / int64(size))
}
