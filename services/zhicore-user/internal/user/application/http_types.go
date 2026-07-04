package application

import (
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/ports"
)

type UserID int64

type AccountID int64

type PublicID string

type UserStatus string

const (
	UserStatusActive      UserStatus = "ACTIVE"
	UserStatusDeactivated UserStatus = "DEACTIVATED"
	UserStatusDeleted     UserStatus = "DELETED"
)

type Profile struct {
	UserID                 UserID
	PublicID               PublicID
	AccountID              AccountID
	Nickname               string
	AvatarFileID           string
	Bio                    string
	StrangerMessageAllowed bool
	Status                 UserStatus
	ProfileVersion         int64
	DeletedReason          string
	DeletedBy              UserID
	DeletedAt              time.Time
	RestoredReason         string
	RestoredBy             UserID
	RestoredAt             time.Time
	DeactivatedAt          time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type ProfileSeed struct {
	UserID                 UserID
	PublicID               PublicID
	AccountID              AccountID
	Nickname               string
	AvatarFileID           string
	Bio                    string
	StrangerMessageAllowed bool
	Status                 UserStatus
	ProfileVersion         int64
	DeletedReason          string
	DeletedBy              UserID
	DeletedAt              time.Time
	RestoredReason         string
	RestoredBy             UserID
	RestoredAt             time.Time
	DeactivatedAt          time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

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

func NewProfile(seed ProfileSeed) (Profile, error) {
	profile, err := domain.NewProfile(domain.ProfileSeed{
		UserID:                 domainUserID(seed.UserID),
		PublicID:               domainPublicID(seed.PublicID),
		AccountID:              domain.AccountID(seed.AccountID),
		Nickname:               seed.Nickname,
		AvatarFileID:           seed.AvatarFileID,
		Bio:                    seed.Bio,
		StrangerMessageAllowed: seed.StrangerMessageAllowed,
		Status:                 domainUserStatus(seed.Status),
		ProfileVersion:         seed.ProfileVersion,
		DeletedReason:          seed.DeletedReason,
		DeletedBy:              domainUserID(seed.DeletedBy),
		DeletedAt:              seed.DeletedAt,
		RestoredReason:         seed.RestoredReason,
		RestoredBy:             domainUserID(seed.RestoredBy),
		RestoredAt:             seed.RestoredAt,
		DeactivatedAt:          seed.DeactivatedAt,
		CreatedAt:              seed.CreatedAt,
		UpdatedAt:              seed.UpdatedAt,
	})
	if err != nil {
		return Profile{}, err
	}
	return profileFromDomain(profile), nil
}

func profileFromDomain(profile domain.Profile) Profile {
	return Profile{
		UserID:                 UserID(profile.UserID),
		PublicID:               PublicID(profile.PublicID),
		AccountID:              AccountID(profile.AccountID),
		Nickname:               profile.Nickname,
		AvatarFileID:           profile.AvatarFileID,
		Bio:                    profile.Bio,
		StrangerMessageAllowed: profile.StrangerMessageAllowed,
		Status:                 UserStatus(profile.Status),
		ProfileVersion:         profile.ProfileVersion,
		DeletedReason:          profile.DeletedReason,
		DeletedBy:              UserID(profile.DeletedBy),
		DeletedAt:              profile.DeletedAt,
		RestoredReason:         profile.RestoredReason,
		RestoredBy:             UserID(profile.RestoredBy),
		RestoredAt:             profile.RestoredAt,
		DeactivatedAt:          profile.DeactivatedAt,
		CreatedAt:              profile.CreatedAt,
		UpdatedAt:              profile.UpdatedAt,
	}
}

func profilesFromDomain(profiles []domain.Profile) []Profile {
	items := make([]Profile, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, profileFromDomain(profile))
	}
	return items
}

func domainUserID(userID UserID) domain.UserID {
	return domain.UserID(userID)
}

func domainPublicID(publicID PublicID) domain.PublicID {
	return domain.PublicID(publicID)
}

func domainUserStatus(status UserStatus) domain.UserStatus {
	return domain.UserStatus(status)
}
