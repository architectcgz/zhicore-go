package domain

import (
	"strings"
	"time"
)

type AccountID int64

type UserID int64

type AccountStatus string

const (
	AccountStatusPendingProfile AccountStatus = "PENDING_PROFILE"
	AccountStatusActive         AccountStatus = "ACTIVE"
	AccountStatusDisabled       AccountStatus = "DISABLED"
	AccountStatusBanned         AccountStatus = "BANNED"
	AccountStatusExpired        AccountStatus = "EXPIRED"
)

type RoleName string

const (
	RoleUser      RoleName = "ROLE_USER"
	RoleAdmin     RoleName = "ROLE_ADMIN"
	RoleModerator RoleName = "ROLE_MODERATOR"
)

type Email struct {
	normalized string
}

func NewEmail(raw string) (Email, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" || !strings.Contains(normalized, "@") {
		return Email{}, ErrEmailInvalid
	}
	return Email{normalized: normalized}, nil
}

func (e Email) Normalized() string {
	return e.normalized
}

type Account struct {
	ID                     AccountID
	UserID                 UserID
	Email                  Email
	Status                 AccountStatus
	PendingProfileNickname string
	SessionVersion         int64
	PrincipalVersion       int64
	LockedUntil            time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func NewPendingAccount(email Email, nickname string, now time.Time) Account {
	return Account{
		Email:                  email,
		Status:                 AccountStatusPendingProfile,
		PendingProfileNickname: nickname,
		SessionVersion:         1,
		PrincipalVersion:       1,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func (a Account) CanLogin(now time.Time) error {
	switch a.Status {
	case AccountStatusDisabled:
		return ErrAccountDisabled
	case AccountStatusBanned:
		return ErrAccountBanned
	}
	// ACTIVE 账号必须已经绑定 User profile；缺失 userId 说明认证事实损坏，不能继续签发登录态。
	if a.Status == AccountStatusActive && a.UserID == 0 {
		return ErrInvalidCredentials
	}
	if !a.LockedUntil.IsZero() && a.LockedUntil.After(now) {
		return ErrAccountLocked
	}
	if a.Status != AccountStatusActive {
		return ErrInvalidCredentials
	}
	return nil
}

type Credential struct {
	AccountID         AccountID
	PasswordHash      string
	CredentialVersion int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewCredential(accountID AccountID, passwordHash string, now time.Time) Credential {
	return Credential{
		AccountID:         accountID,
		PasswordHash:      passwordHash,
		CredentialVersion: 1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}
