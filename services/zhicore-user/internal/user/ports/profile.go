package ports

import (
	"context"
	"errors"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
)

var ErrDependencyUnavailable = errors.New("dependency unavailable")

type ProfileRepository interface {
	FindByAccountID(ctx context.Context, accountID domain.AccountID) (domain.Profile, error)
	// CreateOrGetByAccountID must be safe under unique-key contention and return
	// the stored profile without poisoning the surrounding transaction.
	CreateOrGetByAccountID(ctx context.Context, profile domain.Profile) (domain.Profile, bool, error)
	Update(ctx context.Context, profile domain.Profile) (domain.Profile, error)
	// UpdatePublicProfile owns the atomic profile_version = profile_version + 1 write and must return the stored version.
	UpdatePublicProfile(ctx context.Context, profile domain.Profile) (domain.Profile, error)
	// Status transition commands own the locked/conditional write so application
	// never overwrites a stale pre-transaction snapshot.
	DeactivateByAccountID(ctx context.Context, accountID domain.AccountID, now time.Time) (domain.Profile, bool, error)
	MarkDeleted(ctx context.Context, userID, operatorID domain.UserID, reason string, now time.Time) (domain.Profile, bool, error)
	RestoreDeleted(ctx context.Context, userID, operatorID domain.UserID, reason string, now time.Time) (domain.Profile, bool, error)
}

type ProfileQueryRepository interface {
	GetByUserID(ctx context.Context, userID domain.UserID) (domain.Profile, error)
	GetByPublicID(ctx context.Context, publicID domain.PublicID) (domain.Profile, error)
}

type FileReferenceClient interface {
	EnsureAvatarReferenced(ctx context.Context, fileID string) error
}

type PublicIDGenerator interface {
	Generate(ctx context.Context) (domain.PublicID, error)
}

type OutboxMessage struct {
	EventType     string
	AggregateType string
	AggregateID   string
	OccurredAt    time.Time
	Payload       []byte
}

type OutboxPublisher interface {
	Publish(ctx context.Context, message OutboxMessage) error
}

type TransactionRunner interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type Clock interface {
	Now() time.Time
}

type CacheStore interface {
	Delete(ctx context.Context, keys ...string) error
}
