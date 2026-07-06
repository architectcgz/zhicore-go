package application

import (
	"context"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

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

func rootPublicID(ids ports.CommentIDCodec, comment domain.Comment) PublicCommentID {
	if !comment.IsReply() {
		return ""
	}
	return PublicCommentID(ids.Encode(comment.RootID))
}
