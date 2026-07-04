package domain

import "errors"

var (
	ErrProfileNotFound         = errors.New("profile not found")
	ErrNicknameInvalid         = errors.New("nickname is invalid")
	ErrNicknameTaken           = errors.New("nickname is taken")
	ErrBioInvalid              = errors.New("bio is invalid")
	ErrAvatarInvalid           = errors.New("avatar is invalid")
	ErrUserNotActive           = errors.New("user is not active")
	ErrAccountAlreadyExists    = errors.New("account profile already exists")
	ErrPublicIDInvalid         = errors.New("public id is invalid")
	ErrInvalidStatusTransition = errors.New("invalid user status transition")
	ErrCannotFollowSelf        = errors.New("cannot follow self")
	ErrCannotBlockSelf         = errors.New("cannot block self")
	ErrInteractionBlocked      = errors.New("interaction is blocked")
	ErrCursorInvalid           = errors.New("cursor is invalid")
)
