package domain

import "strings"

type CommentMedia struct {
	ImageFileIDs  []string
	VoiceFileID   string
	VoiceDuration int
}

type CommentMediaInput struct {
	ImageFileIDs  []string
	VoiceFileID   string
	VoiceDuration int
}

func NewCommentMedia(input CommentMediaInput) (CommentMedia, error) {
	images := normalizeFileIDs(input.ImageFileIDs)
	voiceFileID := strings.TrimSpace(input.VoiceFileID)
	if len(images) > maxImageFileCount {
		return CommentMedia{}, ErrCommentMediaInvalid
	}
	if voiceFileID != "" && len(images) > 0 {
		return CommentMedia{}, ErrCommentMediaInvalid
	}
	if voiceFileID == "" && input.VoiceDuration != 0 {
		return CommentMedia{}, ErrCommentMediaInvalid
	}
	if voiceFileID != "" && input.VoiceDuration <= 0 {
		return CommentMedia{}, ErrCommentMediaInvalid
	}
	return CommentMedia{
		ImageFileIDs:  images,
		VoiceFileID:   voiceFileID,
		VoiceDuration: input.VoiceDuration,
	}, nil
}

func (m CommentMedia) Empty() bool {
	return len(m.ImageFileIDs) == 0 && strings.TrimSpace(m.VoiceFileID) == ""
}

func normalizeFileIDs(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
