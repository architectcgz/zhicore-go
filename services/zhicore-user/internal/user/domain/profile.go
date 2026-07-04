package domain

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type UserID int64

type AccountID int64

type PublicID string

type UserStatus string

const (
	UserStatusActive      UserStatus = "ACTIVE"
	UserStatusDeactivated UserStatus = "DEACTIVATED"
	UserStatusDeleted     UserStatus = "DELETED"
	maxNicknameRuneCount             = 15
	maxBioRuneCount                  = 100
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

type ProfileUpdate struct {
	Nickname               string
	AvatarFileID           string
	Bio                    string
	StrangerMessageAllowed bool
}

func NewProfile(seed ProfileSeed) (Profile, error) {
	if seed.AccountID == 0 {
		return Profile{}, ErrProfileNotFound
	}
	publicID := strings.TrimSpace(string(seed.PublicID))
	if publicID == "" {
		return Profile{}, ErrPublicIDInvalid
	}
	nickname, err := normalizeNickname(seed.Nickname)
	if err != nil {
		return Profile{}, err
	}
	bio, err := normalizeBio(seed.Bio)
	if err != nil {
		return Profile{}, err
	}
	status := seed.Status
	if status == "" {
		status = UserStatusActive
	}
	if !isValidStatus(status) {
		return Profile{}, ErrInvalidStatusTransition
	}
	return Profile{
		UserID:                 seed.UserID,
		PublicID:               PublicID(publicID),
		AccountID:              seed.AccountID,
		Nickname:               nickname,
		AvatarFileID:           strings.TrimSpace(seed.AvatarFileID),
		Bio:                    bio,
		StrangerMessageAllowed: seed.StrangerMessageAllowed,
		Status:                 status,
		ProfileVersion:         seed.ProfileVersion,
		DeletedReason:          strings.TrimSpace(seed.DeletedReason),
		DeletedBy:              seed.DeletedBy,
		DeletedAt:              seed.DeletedAt,
		RestoredReason:         strings.TrimSpace(seed.RestoredReason),
		RestoredBy:             seed.RestoredBy,
		RestoredAt:             seed.RestoredAt,
		DeactivatedAt:          seed.DeactivatedAt,
		CreatedAt:              seed.CreatedAt,
		UpdatedAt:              seed.UpdatedAt,
	}, nil
}

func NewProfileForAccount(accountID AccountID, publicID PublicID, username string, now time.Time) (Profile, error) {
	return NewProfile(ProfileSeed{
		PublicID:               publicID,
		AccountID:              accountID,
		Nickname:               username,
		Bio:                    "",
		StrangerMessageAllowed: true,
		Status:                 UserStatusActive,
		ProfileVersion:         0,
		CreatedAt:              now,
		UpdatedAt:              now,
	})
}

func (p Profile) ApplyUpdate(input ProfileUpdate, now time.Time) (Profile, bool, error) {
	if p.Status != UserStatusActive {
		return Profile{}, false, ErrUserNotActive
	}
	nickname, err := normalizeNickname(input.Nickname)
	if err != nil {
		return Profile{}, false, err
	}
	bio, err := normalizeBio(input.Bio)
	if err != nil {
		return Profile{}, false, err
	}
	avatarFileID := strings.TrimSpace(input.AvatarFileID)
	publicChanged := nickname != p.Nickname || avatarFileID != p.AvatarFileID || bio != p.Bio
	p.Nickname = nickname
	p.AvatarFileID = avatarFileID
	p.Bio = bio
	p.StrangerMessageAllowed = input.StrangerMessageAllowed
	p.UpdatedAt = now
	return p, publicChanged, nil
}

func (p Profile) Deactivate(now time.Time) (Profile, bool, error) {
	switch p.Status {
	case UserStatusActive:
		p.Status = UserStatusDeactivated
		p.DeactivatedAt = now
		p.UpdatedAt = now
		return p, true, nil
	case UserStatusDeactivated:
		return p, false, nil
	default:
		return Profile{}, false, ErrInvalidStatusTransition
	}
}

func (p Profile) MarkDeleted(operatorID UserID, reason string, now time.Time) (Profile, bool) {
	if p.Status == UserStatusDeleted {
		return p, false
	}
	p.Status = UserStatusDeleted
	p.DeletedReason = strings.TrimSpace(reason)
	p.DeletedBy = operatorID
	p.DeletedAt = now
	p.UpdatedAt = now
	return p, true
}

func (p Profile) RestoreDeleted(operatorID UserID, reason string, now time.Time) (Profile, bool, error) {
	if p.Status == UserStatusActive {
		return p, false, nil
	}
	if p.Status != UserStatusDeleted {
		return Profile{}, false, ErrInvalidStatusTransition
	}
	p.Status = UserStatusActive
	p.RestoredReason = strings.TrimSpace(reason)
	p.RestoredBy = operatorID
	p.RestoredAt = now
	p.UpdatedAt = now
	return p, true, nil
}

func normalizeNickname(raw string) (string, error) {
	nickname := strings.TrimSpace(raw)
	if nickname == "" || utf8.RuneCountInString(nickname) > maxNicknameRuneCount {
		return "", ErrNicknameInvalid
	}
	if containsDangerousRunes(nickname, false) {
		return "", ErrNicknameInvalid
	}
	return nickname, nil
}

func normalizeBio(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	if utf8.RuneCountInString(raw) > maxBioRuneCount {
		return "", ErrBioInvalid
	}
	if containsDangerousRunes(raw, true) {
		return "", ErrBioInvalid
	}
	return raw, nil
}

func containsDangerousRunes(raw string, allowNewlines bool) bool {
	for _, r := range raw {
		switch r {
		case '<', '>':
			return true
		case '\n':
			// Bio is plain text and only allows LF line breaks; other control
			// whitespace such as CR and tabs stays invalid to avoid hidden input.
			if allowNewlines {
				continue
			}
			return true
		case '\r', '\t':
			return true
		}
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func isValidStatus(status UserStatus) bool {
	switch status {
	case UserStatusActive, UserStatusDeactivated, UserStatusDeleted:
		return true
	default:
		return false
	}
}
