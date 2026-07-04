package domain

type CommentStats struct {
	CommentID  CommentID
	LikeCount  int64
	ReplyCount int64
}

type CommentPostStats struct {
	PostID                PostID
	TotalComments         int64
	TotalTopLevelComments int64
}
