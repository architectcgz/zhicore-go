package domain

import "errors"

var (
	ErrCommentIDInvalid       = errors.New("comment id is invalid")
	ErrCommentContentRequired = errors.New("comment content is required")
	ErrCommentContentTooLong  = errors.New("comment content is too long")
	ErrCommentMediaInvalid    = errors.New("comment media is invalid")
	ErrCommentNotFound        = errors.New("comment not found")
	ErrParentCommentNotFound  = errors.New("parent comment not found")
	ErrRootCommentNotFound    = errors.New("root comment not found")
	ErrPostIDInvalid          = errors.New("post id is invalid")
	ErrUserIDInvalid          = errors.New("user id is invalid")
)
