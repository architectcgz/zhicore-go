package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/ports"
)

const registeredEventType = "auth.account.registered"

type Dependencies struct {
	Accounts      ports.AccountRepository
	Credentials   ports.CredentialRepository
	Roles         ports.RoleRepository
	Sessions      ports.RefreshSessionStore
	Tokens        ports.TokenIssuer
	RefreshTokens ports.RefreshTokenMaterialIssuer
	Hasher        ports.PasswordHasher
	Blacklist     ports.AccessTokenBlacklist
	Cache         ports.AuthCacheStore
	TxRunner      ports.TransactionRunner
	Outbox        ports.OutboxPublisher
	UserProfiles  ports.UserProfileClient
	Clock         ports.Clock
	RateLimiter   ports.RateLimiter
}

type Service struct {
	accounts      ports.AccountRepository
	credentials   ports.CredentialRepository
	roles         ports.RoleRepository
	sessions      ports.RefreshSessionStore
	tokens        ports.TokenIssuer
	refreshTokens ports.RefreshTokenMaterialIssuer
	hasher        ports.PasswordHasher
	blacklist     ports.AccessTokenBlacklist
	cache         ports.AuthCacheStore
	txRunner      ports.TransactionRunner
	outbox        ports.OutboxPublisher
	userProfiles  ports.UserProfileClient
	clock         ports.Clock
	rateLimiter   ports.RateLimiter
}

func NewService(deps Dependencies) (*Service, error) {
	if err := validateDependencies(deps); err != nil {
		return nil, err
	}
	return &Service{
		accounts:      deps.Accounts,
		credentials:   deps.Credentials,
		roles:         deps.Roles,
		sessions:      deps.Sessions,
		tokens:        deps.Tokens,
		refreshTokens: deps.RefreshTokens,
		hasher:        deps.Hasher,
		blacklist:     deps.Blacklist,
		cache:         deps.Cache,
		txRunner:      deps.TxRunner,
		outbox:        deps.Outbox,
		userProfiles:  deps.UserProfiles,
		clock:         deps.Clock,
		rateLimiter:   deps.RateLimiter,
	}, nil
}

type RegisterAccountCommand struct {
	Nickname string
	Email    string
	Password string
}

type RegisterAccountResult struct {
	AccountID domain.AccountID
	UserID    domain.UserID
}

type LoginCommand struct {
	Email    string
	Password string
}

type LoginResult struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	SessionID             string
}

func (s *Service) RegisterAccount(ctx context.Context, cmd RegisterAccountCommand) (RegisterAccountResult, error) {
	now := s.clock.Now()
	email, err := domain.NewEmail(cmd.Email)
	if err != nil {
		return RegisterAccountResult{}, err
	}
	if err := s.rateLimit(ctx, domain.SecurityOperationRegister, email.Normalized()); err != nil {
		return RegisterAccountResult{}, err
	}

	passwordHash, err := s.hasher.HashPassword(ctx, cmd.Password)
	if err != nil {
		return RegisterAccountResult{}, fmt.Errorf("hash password: %w", err)
	}

	var created domain.Account
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		created, err = s.accounts.CreateOrLoadPendingForRegister(txCtx, ports.CreateOrLoadPendingAccountInput{
			Email:    email,
			Nickname: cmd.Nickname,
			Now:      now,
		})
		if err != nil {
			return fmt.Errorf("create or load pending account: %w", err)
		}
		if err := s.credentials.SaveForPendingAccount(txCtx, domain.NewCredential(created.ID, passwordHash, now)); err != nil {
			return fmt.Errorf("save pending credential: %w", err)
		}
		return nil
	}); err != nil {
		return RegisterAccountResult{}, err
	}

	createdProfile, err := s.userProfiles.CreateProfileForAccount(ctx, ports.CreateProfileForAccountInput{
		AccountID: created.ID,
		Nickname:  cmd.Nickname,
	})
	if err != nil {
		return RegisterAccountResult{}, fmt.Errorf("create user profile: %w", err)
	}
	if createdProfile.UserID == 0 {
		return RegisterAccountResult{}, fmt.Errorf("user profile contract violation: zero user id")
	}
	// 事务 B 和 outbox 要记录 profile 真正闭合后的激活时间，不能复用请求开始时间。
	activatedAt := s.clock.Now()

	// 同步闭合 User profile，避免前端收到“注册成功”后立刻遇到登录失败或资料接口 404。
	// `auth.account.registered` 只表达已经 ACTIVE 且拿到 userId 的账号事实，因此必须在 User 初始化成功后再入本地 outbox。
	if err := s.txRunner.WithinTransaction(ctx, func(txCtx context.Context) error {
		role, err := s.roles.DefaultRole(txCtx)
		if err != nil {
			return fmt.Errorf("load default role: %w", err)
		}

		created, err = s.accounts.Activate(txCtx, ports.ActivateAccountInput{
			AccountID:   created.ID,
			UserID:      createdProfile.UserID,
			ActivatedAt: activatedAt,
		})
		if err != nil {
			return fmt.Errorf("activate account: %w", err)
		}
		if err := s.roles.Assign(txCtx, created.ID, role); err != nil {
			return fmt.Errorf("assign default role: %w", err)
		}

		payload, err := json.Marshal(map[string]any{
			"accountId":  created.ID,
			"userId":     createdProfile.UserID,
			"email":      created.Email.Normalized(),
			"occurredAt": activatedAt.UTC().Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("marshal registered event: %w", err)
		}
		if err := s.outbox.Publish(txCtx, ports.OutboxMessage{
			EventType:  registeredEventType,
			OccurredAt: activatedAt,
			Payload:    payload,
		}); err != nil {
			return fmt.Errorf("publish registered outbox: %w", err)
		}
		return nil
	}); err != nil {
		return RegisterAccountResult{}, err
	}

	return RegisterAccountResult{AccountID: created.ID, UserID: created.UserID}, nil
}

func (s *Service) Login(ctx context.Context, cmd LoginCommand) (LoginResult, error) {
	now := s.clock.Now()
	email, err := domain.NewEmail(cmd.Email)
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.rateLimit(ctx, domain.SecurityOperationLogin, email.Normalized()); err != nil {
		return LoginResult{}, err
	}

	account, err := s.accounts.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			return LoginResult{}, domain.ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("find account by email: %w", err)
	}
	if err := account.CanLogin(now); err != nil {
		return LoginResult{}, err
	}

	credential, err := s.credentials.FindByAccountID(ctx, account.ID)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialNotFound) {
			return LoginResult{}, domain.ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("find credential: %w", err)
	}

	ok, err := s.hasher.VerifyPassword(ctx, cmd.Password, credential.PasswordHash)
	if err != nil {
		return LoginResult{}, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return LoginResult{}, domain.ErrInvalidCredentials
	}

	roles, err := s.roles.ListByAccountID(ctx, account.ID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("list roles: %w", err)
	}

	refreshMaterial, err := s.refreshTokens.GenerateLoginMaterial(ctx, ports.GenerateRefreshTokenMaterialInput{
		AccountID: account.ID,
		IssuedAt:  now,
	})
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate refresh token material: %w", err)
	}
	if err := validateRefreshTokenMaterial(refreshMaterial, now); err != nil {
		return LoginResult{}, err
	}

	if err := s.sessions.Create(ctx, ports.CreateRefreshSessionInput{
		AccountID:        account.ID,
		SessionID:        refreshMaterial.SessionID,
		CurrentTokenID:   refreshMaterial.TokenID,
		CurrentTokenHash: refreshMaterial.TokenHash,
		ExpiresAt:        refreshMaterial.ExpiresAt,
		CreatedAt:        now,
	}); err != nil {
		return LoginResult{}, fmt.Errorf("create refresh session: %w", err)
	}

	accessToken, err := s.tokens.IssueLoginAccessToken(ctx, ports.IssueAccessTokenRequest{
		AccountID:        account.ID,
		UserID:           account.UserID,
		AccountStatus:    account.Status,
		Roles:            roles,
		SessionID:        refreshMaterial.SessionID,
		SessionVersion:   account.SessionVersion,
		PrincipalVersion: account.PrincipalVersion,
		IssuedAt:         now,
	})
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue tokens: %w", err)
	}

	return LoginResult{
		AccessToken:           accessToken.Token,
		AccessTokenExpiresAt:  accessToken.ExpiresAt,
		RefreshToken:          refreshMaterial.Plaintext,
		RefreshTokenExpiresAt: refreshMaterial.ExpiresAt,
		SessionID:             refreshMaterial.SessionID,
	}, nil
}

func (s *Service) rateLimit(ctx context.Context, operation domain.SecurityOperation, key string) error {
	if s.rateLimiter == nil {
		return fmt.Errorf("rate limiter dependency is required")
	}
	result, err := s.rateLimiter.Check(ctx, operation, key)
	if err != nil {
		return err
	}
	switch result.Outcome {
	case ports.RateLimitOutcomeAllow, ports.RateLimitOutcomeDegraded:
		return nil
	case "":
		return fmt.Errorf("rate limiter returned empty outcome")
	case ports.RateLimitOutcomeReject:
		return domain.ErrRateLimitExceeded
	case ports.RateLimitOutcomeUnavailable:
		return domain.ErrRateLimitUnavailable
	default:
		return fmt.Errorf("unsupported rate limit outcome: %s", result.Outcome)
	}
}

func validateDependencies(deps Dependencies) error {
	required := []struct {
		name  string
		value any
	}{
		{name: "Accounts", value: deps.Accounts},
		{name: "Credentials", value: deps.Credentials},
		{name: "Roles", value: deps.Roles},
		{name: "Sessions", value: deps.Sessions},
		{name: "Tokens", value: deps.Tokens},
		{name: "RefreshTokens", value: deps.RefreshTokens},
		{name: "Hasher", value: deps.Hasher},
		{name: "TxRunner", value: deps.TxRunner},
		{name: "Outbox", value: deps.Outbox},
		{name: "UserProfiles", value: deps.UserProfiles},
		{name: "Clock", value: deps.Clock},
		{name: "RateLimiter", value: deps.RateLimiter},
	}
	for _, dep := range required {
		if dep.value == nil {
			return fmt.Errorf("%s dependency is required", dep.name)
		}
	}
	return nil
}

func validateRefreshTokenMaterial(material ports.GeneratedRefreshTokenMaterial, now time.Time) error {
	fields := []struct {
		name  string
		value string
	}{
		{name: "session id", value: material.SessionID},
		{name: "token id", value: material.TokenID},
		{name: "token hash", value: material.TokenHash},
		{name: "plaintext", value: material.Plaintext},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("refresh token material contract violation: missing %s", field.name)
		}
	}
	if !material.ExpiresAt.After(now) {
		return fmt.Errorf("refresh token material contract violation: expiresAt must be after issued time")
	}
	return nil
}
