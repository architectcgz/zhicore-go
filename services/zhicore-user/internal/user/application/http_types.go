package application

import (
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/ports"
)

type (
	UserID      = domain.UserID
	PublicID    = domain.PublicID
	UserStatus  = domain.UserStatus
	Profile     = domain.Profile
	ProfileSeed = domain.ProfileSeed
)

var (
	UserStatusActive = domain.UserStatusActive

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

func NewProfile(seed ProfileSeed) (Profile, error) {
	return domain.NewProfile(seed)
}
