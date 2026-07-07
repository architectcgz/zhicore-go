package application

import "time"

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

type UserSimple struct {
	UserID         UserID
	PublicID       PublicID
	Nickname       string
	AvatarFileID   string
	ProfileVersion int64
	Status         UserStatus
}

type BatchUserSimpleResult struct {
	Items          []UserSimple
	MissingUserIDs []UserID
}

type UserAvailability struct {
	UserID    UserID
	Available bool
	Status    UserStatus
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
