package ports

import (
	"context"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/domain"
)

type AccountRepository interface {
	CreateOrLoadPendingForRegister(ctx context.Context, input CreateOrLoadPendingAccountInput) (domain.Account, error)
	FindByEmail(ctx context.Context, email domain.Email) (domain.Account, error)
	Activate(ctx context.Context, input ActivateAccountInput) (domain.Account, error)
}

type CreateOrLoadPendingAccountInput struct {
	Email    domain.Email
	Nickname string
	Now      time.Time
}

type ActivateAccountInput struct {
	AccountID   domain.AccountID
	UserID      domain.UserID
	ActivatedAt time.Time
}

type CredentialRepository interface {
	SaveForPendingAccount(ctx context.Context, credential domain.Credential) error
	FindByAccountID(ctx context.Context, accountID domain.AccountID) (domain.Credential, error)
}

type RoleRepository interface {
	DefaultRole(ctx context.Context) (domain.RoleName, error)
	Assign(ctx context.Context, accountID domain.AccountID, role domain.RoleName) error
	ListByAccountID(ctx context.Context, accountID domain.AccountID) ([]domain.RoleName, error)
}

type RefreshSessionStore interface {
	Create(ctx context.Context, input CreateRefreshSessionInput) error
}

type CreateRefreshSessionInput struct {
	AccountID        domain.AccountID
	SessionID        string
	CurrentTokenID   string
	CurrentTokenHash string
	ExpiresAt        time.Time
	CreatedAt        time.Time
}

type CreateProfileForAccountInput struct {
	AccountID domain.AccountID
	Nickname  string
}

type CreatedUserProfile struct {
	UserID domain.UserID
}

type UserProfileClient interface {
	CreateProfileForAccount(ctx context.Context, input CreateProfileForAccountInput) (CreatedUserProfile, error)
}

type TransactionRunner interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type Clock interface {
	Now() time.Time
}
