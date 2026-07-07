package ports

import (
	"context"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/domain"
)

type GenerateRefreshTokenMaterialInput struct {
	AccountID         domain.AccountID
	IssuedAt          time.Time
	PersistencePolicy domain.RefreshSessionPersistencePolicy
	TTL               time.Duration
}

type GeneratedRefreshTokenMaterial struct {
	SessionID string
	TokenID   string
	Plaintext string
	TokenHash string
	ExpiresAt time.Time
}

type RefreshTokenMaterialIssuer interface {
	GenerateLoginMaterial(ctx context.Context, input GenerateRefreshTokenMaterialInput) (GeneratedRefreshTokenMaterial, error)
}

type IssueAccessTokenRequest struct {
	AccountID        domain.AccountID
	UserID           domain.UserID
	AccountStatus    domain.AccountStatus
	Roles            []domain.RoleName
	SessionID        string
	SessionVersion   int64
	PrincipalVersion int64
	IssuedAt         time.Time
}

type IssuedAccessToken struct {
	Token     string
	JTI       string
	ExpiresAt time.Time
}

type TokenIssuer interface {
	IssueLoginAccessToken(ctx context.Context, request IssueAccessTokenRequest) (IssuedAccessToken, error)
}

type PasswordHasher interface {
	HashPassword(ctx context.Context, password string) (string, error)
	VerifyPassword(ctx context.Context, password string, passwordHash string) (bool, error)
}

type AccessTokenBlacklist interface {
	Blacklist(ctx context.Context, tokenID string, expiresAt time.Time) error
}

type PrincipalSnapshot struct {
	AccountID        domain.AccountID
	AccountStatus    domain.AccountStatus
	Roles            []domain.RoleName
	SessionVersion   int64
	PrincipalVersion int64
}

type AuthCacheStore interface {
	StorePrincipal(ctx context.Context, principal PrincipalSnapshot) error
	MarkSessionRevoked(ctx context.Context, sessionID string, expiresAt time.Time) error
	SetSessionVersion(ctx context.Context, accountID domain.AccountID, version int64) error
	SetPrincipalVersion(ctx context.Context, accountID domain.AccountID, version int64) error
}

type RateLimiter interface {
	Check(ctx context.Context, operation domain.SecurityOperation, key string) (RateLimitResult, error)
}

type RateLimitOutcome string

const (
	RateLimitOutcomeAllow       RateLimitOutcome = "ALLOW"
	RateLimitOutcomeReject      RateLimitOutcome = "REJECT"
	RateLimitOutcomeDegraded    RateLimitOutcome = "DEGRADED"
	RateLimitOutcomeUnavailable RateLimitOutcome = "UNAVAILABLE"
)

type RateLimitResult struct {
	Outcome RateLimitOutcome
}
