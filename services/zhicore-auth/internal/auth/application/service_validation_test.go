package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/ports"
)

func TestNewServiceRejectsMissingRequiredDependencies(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 30, 0, 0, time.UTC)
	base := Dependencies{
		Accounts: &fakeAccountRepository{}, Credentials: &fakeCredentialRepository{}, Roles: &fakeRoleRepository{},
		Sessions: &fakeRefreshSessionStore{}, Tokens: &fakeTokenIssuer{}, RefreshTokens: &fakeRefreshTokenMaterialIssuer{},
		Hasher: &fakePasswordHasher{}, TxRunner: &fakeTransactionRunner{}, Outbox: &fakeOutboxPublisher{},
		UserProfiles: &fakeUserProfileClient{}, Clock: fixedClock{now: now}, RateLimiter: allowRateLimiter(),
	}
	tests := []struct {
		name  string
		apply func(*Dependencies)
		want  string
	}{
		{name: "accounts", apply: func(d *Dependencies) { d.Accounts = nil }, want: "Accounts"},
		{name: "credentials", apply: func(d *Dependencies) { d.Credentials = nil }, want: "Credentials"},
		{name: "roles", apply: func(d *Dependencies) { d.Roles = nil }, want: "Roles"},
		{name: "sessions", apply: func(d *Dependencies) { d.Sessions = nil }, want: "Sessions"},
		{name: "tokens", apply: func(d *Dependencies) { d.Tokens = nil }, want: "Tokens"},
		{name: "refresh tokens", apply: func(d *Dependencies) { d.RefreshTokens = nil }, want: "RefreshTokens"},
		{name: "hasher", apply: func(d *Dependencies) { d.Hasher = nil }, want: "Hasher"},
		{name: "tx runner", apply: func(d *Dependencies) { d.TxRunner = nil }, want: "TxRunner"},
		{name: "outbox", apply: func(d *Dependencies) { d.Outbox = nil }, want: "Outbox"},
		{name: "user profiles", apply: func(d *Dependencies) { d.UserProfiles = nil }, want: "UserProfiles"},
		{name: "clock", apply: func(d *Dependencies) { d.Clock = nil }, want: "Clock"},
		{name: "rate limiter", apply: func(d *Dependencies) { d.RateLimiter = nil }, want: "RateLimiter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := base
			tt.apply(&deps)
			_, err := NewService(deps)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("NewService() error = %v, want mention %q", err, tt.want)
			}
		})
	}
}

func TestNewServiceAllowsNilTaskOneOptionalDependencies(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 40, 0, 0, time.UTC)
	service, err := NewService(Dependencies{
		Accounts: &fakeAccountRepository{}, Credentials: &fakeCredentialRepository{}, Roles: &fakeRoleRepository{},
		Sessions: &fakeRefreshSessionStore{}, Tokens: &fakeTokenIssuer{}, RefreshTokens: &fakeRefreshTokenMaterialIssuer{},
		Hasher:   &fakePasswordHasher{hash: "hashed-password"},
		TxRunner: &fakeTransactionRunner{}, Outbox: &fakeOutboxPublisher{}, UserProfiles: &fakeUserProfileClient{}, Clock: fixedClock{now: now},
		RateLimiter: allowRateLimiter(),
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if service == nil {
		t.Fatal("NewService() returned nil service")
	}
}

func TestRegisterAccountRejectsEmptyRateLimiterOutcome(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 41, 0, 0, time.UTC)
	service, err := NewService(Dependencies{
		Accounts: &fakeAccountRepository{}, Credentials: &fakeCredentialRepository{}, Roles: &fakeRoleRepository{},
		Sessions: &fakeRefreshSessionStore{}, Tokens: &fakeTokenIssuer{}, RefreshTokens: &fakeRefreshTokenMaterialIssuer{},
		Hasher:   &fakePasswordHasher{hash: "hashed-password"},
		TxRunner: &fakeTransactionRunner{}, Outbox: &fakeOutboxPublisher{}, UserProfiles: &fakeUserProfileClient{}, Clock: fixedClock{now: now},
		RateLimiter: &fakeRateLimiter{result: ports.RateLimitResult{}},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, err := service.RegisterAccount(context.Background(), RegisterAccountCommand{Nickname: "Alice", Email: "user@example.com", Password: "Password123"}); err == nil {
		t.Fatal("RegisterAccount() error = nil, want empty rate-limit outcome failure")
	}
}
