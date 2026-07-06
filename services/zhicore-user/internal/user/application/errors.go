package application

import (
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/ports"
)

var (
	ErrDependencyUnavailable = ports.ErrDependencyUnavailable

	ErrProfileNotFound    = domain.ErrProfileNotFound
	ErrNicknameInvalid    = domain.ErrNicknameInvalid
	ErrNicknameTaken      = domain.ErrNicknameTaken
	ErrBioInvalid         = domain.ErrBioInvalid
	ErrAvatarInvalid      = domain.ErrAvatarInvalid
	ErrUserNotActive      = domain.ErrUserNotActive
	ErrPublicIDInvalid    = domain.ErrPublicIDInvalid
	ErrCannotFollowSelf   = domain.ErrCannotFollowSelf
	ErrCannotBlockSelf    = domain.ErrCannotBlockSelf
	ErrInteractionBlocked = domain.ErrInteractionBlocked
	ErrCursorInvalid      = domain.ErrCursorInvalid
)
