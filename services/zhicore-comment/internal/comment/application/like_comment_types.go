package application

import "time"

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
