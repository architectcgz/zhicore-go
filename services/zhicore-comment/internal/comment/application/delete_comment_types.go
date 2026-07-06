package application

import "time"

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
