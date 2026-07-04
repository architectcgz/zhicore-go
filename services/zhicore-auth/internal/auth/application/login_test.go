package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/ports"
)

func TestLoginCreatesSessionFromGeneratedRefreshMaterialBeforeIssuingAccessToken(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 30, 0, 0, time.UTC)
	trace := &callTrace{}
	account := domain.Account{
		ID:               42,
		UserID:           501,
		Email:            mustEmail(t, "user@example.com"),
		Status:           domain.AccountStatusActive,
		SessionVersion:   3,
		PrincipalVersion: 5,
	}
	accountRepo := &fakeAccountRepository{found: account}
	credentialRepo := &fakeCredentialRepository{stored: domain.Credential{AccountID: 42, PasswordHash: "stored-hash"}}
	roleRepo := &fakeRoleRepository{roles: []domain.RoleName{domain.RoleUser}}
	refreshTokens := &fakeRefreshTokenMaterialIssuer{
		trace: trace,
		material: ports.GeneratedRefreshTokenMaterial{
			SessionID: "session-1", TokenID: "refresh-token-id", Plaintext: "refresh-token-plaintext",
			TokenHash: "refresh-token-hash", ExpiresAt: now.Add(30 * 24 * time.Hour),
		},
	}
	sessionStore := &fakeRefreshSessionStore{trace: trace}
	tokenIssuer := &fakeTokenIssuer{
		trace: trace,
		token: ports.IssuedAccessToken{Token: "access-token", JTI: "access-jti", ExpiresAt: now.Add(2 * time.Hour)},
	}

	service := mustNewService(t, Dependencies{
		Accounts: accountRepo, Credentials: credentialRepo, Roles: roleRepo, Sessions: sessionStore,
		Tokens: tokenIssuer, RefreshTokens: refreshTokens, Hasher: &fakePasswordHasher{verifyOK: true},
		Blacklist: &fakeAccessTokenBlacklist{}, Cache: &fakeAuthCacheStore{}, TxRunner: &fakeTransactionRunner{},
		Outbox: &fakeOutboxPublisher{}, UserProfiles: &fakeUserProfileClient{}, Clock: fixedClock{now: now},
		RateLimiter: allowRateLimiter(),
	})

	result, err := service.Login(context.Background(), LoginCommand{Email: "user@example.com", Password: "Password123"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if refreshTokens.issueOrder == 0 || sessionStore.createOrder == 0 || tokenIssuer.issueOrder == 0 {
		t.Fatalf("refresh=%d session=%d access=%d, want all set", refreshTokens.issueOrder, sessionStore.createOrder, tokenIssuer.issueOrder)
	}
	if refreshTokens.issueOrder >= sessionStore.createOrder || sessionStore.createOrder >= tokenIssuer.issueOrder {
		t.Fatalf("refresh/session/access order mismatch: refresh=%d session=%d access=%d", refreshTokens.issueOrder, sessionStore.createOrder, tokenIssuer.issueOrder)
	}
	if sessionStore.createInput.SessionID != refreshTokens.material.SessionID || sessionStore.createInput.CurrentTokenID != refreshTokens.material.TokenID || sessionStore.createInput.CurrentTokenHash != refreshTokens.material.TokenHash {
		t.Fatalf("stored session metadata = %#v, material = %#v", sessionStore.createInput, refreshTokens.material)
	}
	if tokenIssuer.request.AccountID != 42 || tokenIssuer.request.UserID != 501 || tokenIssuer.request.SessionID != "session-1" {
		t.Fatalf("token issuer request = %#v", tokenIssuer.request)
	}
	if len(tokenIssuer.request.Roles) != 1 || tokenIssuer.request.Roles[0] != domain.RoleUser {
		t.Fatalf("token issuer roles = %#v, want [ROLE_USER]", tokenIssuer.request.Roles)
	}
	if result.AccessToken != "access-token" || result.RefreshToken != refreshTokens.material.Plaintext || result.SessionID != "session-1" {
		t.Fatalf("login result = %#v", result)
	}
	if !result.AccessTokenExpiresAt.Equal(now.Add(2*time.Hour)) || !result.RefreshTokenExpiresAt.Equal(refreshTokens.material.ExpiresAt) {
		t.Fatalf("login expirations = %#v", result)
	}
}

func TestLoginMapsMissingAccountCredentialAndWrongPasswordToInvalidCredentials(t *testing.T) {
	now := time.Date(2026, 7, 4, 11, 0, 0, 0, time.UTC)
	activeAccount := domain.Account{ID: 9, UserID: 509, Email: mustEmail(t, "user@example.com"), Status: domain.AccountStatusActive}
	tests := []struct {
		name           string
		accountRepo    *fakeAccountRepository
		credential     domain.Credential
		verifyOK       bool
		wantCredential bool
		wantVerify     bool
	}{
		{name: "account not found", accountRepo: &fakeAccountRepository{}, verifyOK: true},
		{name: "credential not found", accountRepo: &fakeAccountRepository{found: activeAccount}, verifyOK: true, wantCredential: true},
		{name: "password mismatch", accountRepo: &fakeAccountRepository{found: activeAccount}, credential: domain.Credential{AccountID: activeAccount.ID, PasswordHash: "stored-hash"}, wantCredential: true, wantVerify: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentialRepo := &fakeCredentialRepository{stored: tt.credential}
			hasher := &fakePasswordHasher{verifyOK: tt.verifyOK}
			service := mustNewService(t, Dependencies{
				Accounts: tt.accountRepo, Credentials: credentialRepo, Roles: &fakeRoleRepository{roles: []domain.RoleName{domain.RoleUser}},
				Sessions: &fakeRefreshSessionStore{}, Tokens: &fakeTokenIssuer{}, RefreshTokens: &fakeRefreshTokenMaterialIssuer{}, Hasher: hasher,
				Blacklist: &fakeAccessTokenBlacklist{}, Cache: &fakeAuthCacheStore{}, TxRunner: &fakeTransactionRunner{}, Outbox: &fakeOutboxPublisher{}, UserProfiles: &fakeUserProfileClient{},
				Clock: fixedClock{now: now}, RateLimiter: allowRateLimiter(),
			})
			_, err := service.Login(context.Background(), LoginCommand{Email: "user@example.com", Password: "Password123"})
			if !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want %v", err, domain.ErrInvalidCredentials)
			}
			if credentialRepo.findCalled != tt.wantCredential {
				t.Fatalf("credential lookup called = %v, want %v", credentialRepo.findCalled, tt.wantCredential)
			}
			if hasher.verifyCalled != tt.wantVerify {
				t.Fatalf("password verify called = %v, want %v", hasher.verifyCalled, tt.wantVerify)
			}
		})
	}
}

func TestLoginReturnsErrorWhenSessionMetadataCreationFails(t *testing.T) {
	now := time.Date(2026, 7, 4, 11, 30, 0, 0, time.UTC)
	account := domain.Account{ID: 42, UserID: 501, Email: mustEmail(t, "user@example.com"), Status: domain.AccountStatusActive}
	refreshTokens := &fakeRefreshTokenMaterialIssuer{
		trace: &callTrace{},
		material: ports.GeneratedRefreshTokenMaterial{
			SessionID: "session-1", TokenID: "refresh-token-id", Plaintext: "refresh-token",
			TokenHash: "refresh-token-hash", ExpiresAt: now.Add(30 * 24 * time.Hour),
		},
	}
	sessionStore := &fakeRefreshSessionStore{trace: refreshTokens.trace, createErr: errors.New("session store unavailable")}
	tokenIssuer := &fakeTokenIssuer{trace: sessionStore.trace}

	service := mustNewService(t, Dependencies{
		Accounts:    &fakeAccountRepository{found: account},
		Credentials: &fakeCredentialRepository{stored: domain.Credential{AccountID: account.ID, PasswordHash: "stored-hash"}},
		Roles:       &fakeRoleRepository{roles: []domain.RoleName{domain.RoleUser}}, Sessions: sessionStore, Tokens: tokenIssuer, RefreshTokens: refreshTokens,
		Hasher: &fakePasswordHasher{verifyOK: true}, Blacklist: &fakeAccessTokenBlacklist{}, Cache: &fakeAuthCacheStore{}, TxRunner: &fakeTransactionRunner{},
		Outbox: &fakeOutboxPublisher{}, UserProfiles: &fakeUserProfileClient{}, Clock: fixedClock{now: now}, RateLimiter: allowRateLimiter(),
	})

	_, err := service.Login(context.Background(), LoginCommand{Email: "user@example.com", Password: "Password123"})
	if err == nil {
		t.Fatal("Login() error = nil, want failure")
	}
	if tokenIssuer.issueOrder != 0 {
		t.Fatalf("token issuer called with order %d after session creation failed", tokenIssuer.issueOrder)
	}
}

func TestLoginRejectsInvalidRefreshMaterialContract(t *testing.T) {
	now := time.Date(2026, 7, 4, 11, 45, 0, 0, time.UTC)
	tests := []struct {
		name     string
		material ports.GeneratedRefreshTokenMaterial
		wantPart string
	}{
		{
			name: "missing session id",
			material: ports.GeneratedRefreshTokenMaterial{
				TokenID: "token-id", Plaintext: "refresh-token", TokenHash: "token-hash", ExpiresAt: now.Add(time.Hour),
			},
			wantPart: "session id",
		},
		{
			name: "missing token id",
			material: ports.GeneratedRefreshTokenMaterial{
				SessionID: "session-1", Plaintext: "refresh-token", TokenHash: "token-hash", ExpiresAt: now.Add(time.Hour),
			},
			wantPart: "token id",
		},
		{
			name: "missing token hash",
			material: ports.GeneratedRefreshTokenMaterial{
				SessionID: "session-1", TokenID: "token-id", Plaintext: "refresh-token", ExpiresAt: now.Add(time.Hour),
			},
			wantPart: "token hash",
		},
		{
			name: "missing plaintext",
			material: ports.GeneratedRefreshTokenMaterial{
				SessionID: "session-1", TokenID: "token-id", TokenHash: "token-hash", ExpiresAt: now.Add(time.Hour),
			},
			wantPart: "plaintext",
		},
		{
			name: "expired material",
			material: ports.GeneratedRefreshTokenMaterial{
				SessionID: "session-1", TokenID: "token-id", Plaintext: "refresh-token", TokenHash: "token-hash", ExpiresAt: now,
			},
			wantPart: "expiresat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace := &callTrace{}
			sessionStore := &fakeRefreshSessionStore{trace: trace}
			tokenIssuer := &fakeTokenIssuer{trace: trace}
			service := mustNewService(t, Dependencies{
				Accounts: &fakeAccountRepository{found: domain.Account{
					ID:               42,
					UserID:           501,
					Email:            mustEmail(t, "user@example.com"),
					Status:           domain.AccountStatusActive,
					SessionVersion:   3,
					PrincipalVersion: 5,
				}},
				Credentials: &fakeCredentialRepository{stored: domain.Credential{AccountID: 42, PasswordHash: "stored-hash"}},
				Roles:       &fakeRoleRepository{roles: []domain.RoleName{domain.RoleUser}},
				Sessions:    sessionStore,
				Tokens:      tokenIssuer,
				RefreshTokens: &fakeRefreshTokenMaterialIssuer{
					trace:    trace,
					material: tt.material,
				},
				Hasher:   &fakePasswordHasher{verifyOK: true},
				TxRunner: &fakeTransactionRunner{}, Outbox: &fakeOutboxPublisher{}, UserProfiles: &fakeUserProfileClient{},
				Clock: fixedClock{now: now}, RateLimiter: allowRateLimiter(),
			})

			_, err := service.Login(context.Background(), LoginCommand{Email: "user@example.com", Password: "Password123"})
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tt.wantPart) {
				t.Fatalf("Login() error = %v, want mention %q", err, tt.wantPart)
			}
			if sessionStore.createOrder != 0 {
				t.Fatalf("session store called with order %d for invalid material", sessionStore.createOrder)
			}
			if tokenIssuer.issueOrder != 0 {
				t.Fatalf("token issuer called with order %d for invalid material", tokenIssuer.issueOrder)
			}
		})
	}
}

func TestLoginRejectsAccountsThatCannotAuthenticate(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		account domain.Account
		wantErr error
	}{
		{name: "disabled account", account: domain.Account{ID: 1, Email: mustEmail(t, "disabled@example.com"), Status: domain.AccountStatusDisabled}, wantErr: domain.ErrAccountDisabled},
		{name: "banned account", account: domain.Account{ID: 2, Email: mustEmail(t, "banned@example.com"), Status: domain.AccountStatusBanned}, wantErr: domain.ErrAccountBanned},
		{name: "locked account", account: domain.Account{ID: 3, UserID: 503, Email: mustEmail(t, "locked@example.com"), Status: domain.AccountStatusActive, LockedUntil: now.Add(15 * time.Minute)}, wantErr: domain.ErrAccountLocked},
		{name: "active account without user id", account: domain.Account{ID: 4, Email: mustEmail(t, "missing-user@example.com"), Status: domain.AccountStatusActive}, wantErr: domain.ErrInvalidCredentials},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := mustNewService(t, Dependencies{
				Accounts:    &fakeAccountRepository{found: tt.account},
				Credentials: &fakeCredentialRepository{stored: domain.Credential{AccountID: tt.account.ID, PasswordHash: "stored-hash"}},
				Roles:       &fakeRoleRepository{roles: []domain.RoleName{domain.RoleUser}}, Sessions: &fakeRefreshSessionStore{}, Tokens: &fakeTokenIssuer{}, RefreshTokens: &fakeRefreshTokenMaterialIssuer{},
				Hasher: &fakePasswordHasher{verifyOK: true}, Blacklist: &fakeAccessTokenBlacklist{}, Cache: &fakeAuthCacheStore{}, TxRunner: &fakeTransactionRunner{},
				Outbox: &fakeOutboxPublisher{}, UserProfiles: &fakeUserProfileClient{}, Clock: fixedClock{now: now}, RateLimiter: allowRateLimiter(),
			})
			_, err := service.Login(context.Background(), LoginCommand{Email: tt.account.Email.Normalized(), Password: "Password123"})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Login() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
