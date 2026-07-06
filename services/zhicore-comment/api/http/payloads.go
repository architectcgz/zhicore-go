package httpapi

type createCommentReq struct {
	Content         string   `json:"content"`
	ParentCommentID string   `json:"parentCommentId"`
	ImageFileIDs    []string `json:"imageFileIds"`
	VoiceFileID     string   `json:"voiceFileId"`
	VoiceDuration   int      `json:"voiceDuration"`
}

type createCommentResp struct {
	PostID          string `json:"postId"`
	CommentID       string `json:"commentId"`
	RootCommentID   string `json:"rootCommentId,omitempty"`
	ParentCommentID string `json:"parentCommentId,omitempty"`
	CreatedAt       string `json:"createdAt"`
}

type topLevelCommentPageResp struct {
	Items                 []commentItemResp `json:"items"`
	Page                  int               `json:"page"`
	Size                  int               `json:"size"`
	TotalComments         int64             `json:"totalComments"`
	TotalTopLevelComments int64             `json:"totalTopLevelComments"`
	Pages                 int               `json:"pages"`
}

type commentPageResp struct {
	Items []commentItemResp `json:"items"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
	Total int64             `json:"total"`
	Pages int               `json:"pages"`
}

type commentItemResp struct {
	PostID          string            `json:"postId"`
	CommentID       string            `json:"commentId"`
	RootCommentID   string            `json:"rootCommentId,omitempty"`
	ParentCommentID string            `json:"parentCommentId,omitempty"`
	Author          authorSummaryResp `json:"author"`
	Content         string            `json:"content,omitempty"`
	ImageFileIDs    []string          `json:"imageFileIds,omitempty"`
	VoiceFileID     string            `json:"voiceFileId,omitempty"`
	VoiceDuration   int               `json:"voiceDuration,omitempty"`
	Status          string            `json:"status"`
	Stats           commentStatsResp  `json:"stats"`
	Viewer          *viewerStateResp  `json:"viewer,omitempty"`
	CreatedAt       string            `json:"createdAt"`
	UpdatedAt       string            `json:"updatedAt"`
}

type authorSummaryResp struct {
	PublicID     string `json:"publicId,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	AvatarFileID string `json:"avatarFileId,omitempty"`
	AvatarURL    string `json:"avatarUrl,omitempty"`
	Unavailable  bool   `json:"unavailable,omitempty"`
}

type commentStatsResp struct {
	LikeCount  int64 `json:"likeCount"`
	ReplyCount int64 `json:"replyCount"`
}

type viewerStateResp struct {
	Liked bool `json:"liked"`
}

type adminDeleteCommentReq struct {
	Reason string `json:"reason"`
}

type deleteCommentResp struct {
	PostID         string `json:"postId"`
	CommentID      string `json:"commentId"`
	RootCommentID  string `json:"rootCommentId,omitempty"`
	DeletedAt      string `json:"deletedAt"`
	DeletedByRole  string `json:"deletedByRole"`
	AffectedCount  int    `json:"affectedCount"`
	AlreadyDeleted bool   `json:"alreadyDeleted,omitempty"`
}

type likeCommentResp struct {
	PostID     string `json:"postId"`
	CommentID  string `json:"commentId"`
	Liked      bool   `json:"liked"`
	Changed    bool   `json:"changed"`
	OccurredAt string `json:"occurredAt"`
}

type likeStatusResp struct {
	PostID    string `json:"postId"`
	CommentID string `json:"commentId"`
	Liked     bool   `json:"liked"`
}
