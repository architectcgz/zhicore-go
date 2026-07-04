package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxCommentContentRunes = 2000
	maxImageFileCount      = 9
)

type CommentID int64
type PublicCommentID string
type PostID string
type ContentInternalID int64
type UserID int64
type CommentStatus string
type CommentSort string

const (
	CommentStatusNormal  CommentStatus = "NORMAL"
	CommentStatusDeleted CommentStatus = "DELETED"

	CommentSortRecommended CommentSort = "RECOMMENDED"
	CommentSortHot         CommentSort = "HOT"
	CommentSortTime        CommentSort = "TIME"
)

type Comment struct {
	ID                CommentID
	PostID            PostID
	ContentInternalID ContentInternalID
	AuthorID          UserID
	RootID            CommentID
	ParentID          CommentID
	Content           string
	Media             CommentMedia
	Status            CommentStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CommentSeed struct {
	ID                CommentID
	PostID            PostID
	ContentInternalID ContentInternalID
	AuthorID          UserID
	RootID            CommentID
	ParentID          CommentID
	Content           string
	ImageFileIDs      []string
	VoiceFileID       string
	VoiceDuration     int
	Status            CommentStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewComment(seed CommentSeed) (Comment, error) {
	if strings.TrimSpace(string(seed.PostID)) == "" {
		return Comment{}, ErrPostIDInvalid
	}
	if seed.AuthorID <= 0 {
		return Comment{}, ErrUserIDInvalid
	}
	status := seed.Status
	if status == "" {
		status = CommentStatusNormal
	}
	if status != CommentStatusNormal && status != CommentStatusDeleted {
		return Comment{}, ErrCommentNotFound
	}
	content, media, err := NewCommentBody(seed.Content, CommentMediaInput{
		ImageFileIDs:  seed.ImageFileIDs,
		VoiceFileID:   seed.VoiceFileID,
		VoiceDuration: seed.VoiceDuration,
	})
	if err != nil {
		return Comment{}, err
	}
	return Comment{
		ID:                seed.ID,
		PostID:            PostID(strings.TrimSpace(string(seed.PostID))),
		ContentInternalID: seed.ContentInternalID,
		AuthorID:          seed.AuthorID,
		RootID:            seed.RootID,
		ParentID:          seed.ParentID,
		Content:           content,
		Media:             media,
		Status:            status,
		CreatedAt:         seed.CreatedAt,
		UpdatedAt:         seed.UpdatedAt,
	}, nil
}

func NewTopLevelDraft(postID PostID, contentInternalID ContentInternalID, authorID UserID, content string, media CommentMediaInput, now time.Time) (Comment, error) {
	return NewComment(CommentSeed{
		PostID:            postID,
		ContentInternalID: contentInternalID,
		AuthorID:          authorID,
		Content:           content,
		ImageFileIDs:      media.ImageFileIDs,
		VoiceFileID:       media.VoiceFileID,
		VoiceDuration:     media.VoiceDuration,
		Status:            CommentStatusNormal,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
}

func NewReplyDraft(postID PostID, contentInternalID ContentInternalID, authorID UserID, root Comment, parent Comment, content string, media CommentMediaInput, now time.Time) (Comment, error) {
	if root.Status != CommentStatusNormal || !root.IsTopLevel() || root.PostID != postID {
		return Comment{}, ErrRootCommentNotFound
	}
	if parent.Status != CommentStatusNormal || parent.PostID != postID {
		return Comment{}, ErrParentCommentNotFound
	}
	if parent.IsTopLevel() && root.ID != parent.ID {
		return Comment{}, ErrParentCommentNotFound
	}
	if parent.IsReply() && parent.RootID != root.ID {
		return Comment{}, ErrParentCommentNotFound
	}
	return NewComment(CommentSeed{
		PostID:            postID,
		ContentInternalID: contentInternalID,
		AuthorID:          authorID,
		RootID:            root.ID,
		ParentID:          parent.ID,
		Content:           content,
		ImageFileIDs:      media.ImageFileIDs,
		VoiceFileID:       media.VoiceFileID,
		VoiceDuration:     media.VoiceDuration,
		Status:            CommentStatusNormal,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
}

func (c Comment) IsTopLevel() bool {
	return c.RootID == 0 && c.ParentID == 0
}

func (c Comment) IsReply() bool {
	return c.RootID != 0 && c.ParentID != 0
}

func NewCommentBody(rawContent string, rawMedia CommentMediaInput) (string, CommentMedia, error) {
	content := strings.TrimSpace(rawContent)
	media, err := NewCommentMedia(rawMedia)
	if err != nil {
		return "", CommentMedia{}, err
	}
	if content == "" && media.Empty() {
		return "", CommentMedia{}, ErrCommentContentRequired
	}
	if utf8.RuneCountInString(content) > maxCommentContentRunes {
		return "", CommentMedia{}, ErrCommentContentTooLong
	}
	return content, media, nil
}
