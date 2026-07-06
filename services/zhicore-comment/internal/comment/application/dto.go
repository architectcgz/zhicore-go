package application

import (
	"time"
)

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
