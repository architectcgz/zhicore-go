package application

import "time"

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
