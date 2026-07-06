package domain

import "errors"

var (
	ErrPostNotFound         = errors.New("post not found")
	ErrForbidden            = errors.New("forbidden")
	ErrPostAlreadyPublished = errors.New("post already published")
	ErrPostNotPublished     = errors.New("post not published")
	ErrPostDeleted          = errors.New("post deleted")
	ErrTitleRequired        = errors.New("title required")
	ErrTitleTooLong         = errors.New("title too long")
	ErrBodyRequired         = errors.New("body required")
	ErrBodyTooShort         = errors.New("body too short")
	ErrDraftConflict        = errors.New("draft conflict")
	ErrBodyUnavailable      = errors.New("body unavailable")
	ErrBodyInconsistent     = errors.New("body inconsistent")
)
