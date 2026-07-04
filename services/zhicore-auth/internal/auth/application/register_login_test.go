package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-auth/internal/auth/ports"
)

func TestRegisterAccountCreatesOrLoadsPendingAccountThenActivatesWithDefaultRoleAndRegisteredOutbox(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	trace := &callTrace{}
	accountRepo := &fakeAccountRepository{trace: trace}
	credentialRepo := &fakeCredentialRepository{trace: trace}
	roleRepo := &fakeRoleRepository{defaultRole: domain.RoleUser, trace: trace}
	outbox := &fakeOutboxPublisher{trace: trace}
	refreshTokens := &fakeRefreshTokenMaterialIssuer{}
	userProfiles := &fakeUserProfileClient{
		trace: trace,
		result: ports.CreatedUserProfile{
			UserID: 7001,
		},
	}
	txRunner := &fakeTransactionRunner{}

	service := mustNewService(t, Dependencies{
		Accounts:      accountRepo,
		Credentials:   credentialRepo,
		Roles:         roleRepo,
		Sessions:      &fakeRefreshSessionStore{},
		Tokens:        &fakeTokenIssuer{},
		RefreshTokens: refreshTokens,
		Hasher:        &fakePasswordHasher{hash: "hashed-password"},
		Blacklist:     &fakeAccessTokenBlacklist{},
		Cache:         &fakeAuthCacheStore{},
		TxRunner:      txRunner,
		Outbox:        outbox,
		UserProfiles:  userProfiles,
		Clock:         fixedClock{now: now},
		RateLimiter:   allowRateLimiter(),
	})

	result, err := service.RegisterAccount(context.Background(), RegisterAccountCommand{
		Nickname: "Alice",
		Email:    "User@Example.com",
		Password: "Password123",
	})
	if err != nil {
		t.Fatalf("RegisterAccount() error = %v", err)
	}

	if txRunner.calledCount != 2 {
		t.Fatalf("transaction count = %d, want 2", txRunner.calledCount)
	}
	if accountRepo.createOrLoadInput.Email.Normalized() != "user@example.com" {
		t.Fatalf("create/load email = %q, want user@example.com", accountRepo.createOrLoadInput.Email.Normalized())
	}
	if accountRepo.createOrLoadInput.Nickname != "Alice" {
		t.Fatalf("create/load nickname = %q, want Alice", accountRepo.createOrLoadInput.Nickname)
	}
	if accountRepo.createdOrLoaded.Status != domain.AccountStatusPendingProfile {
		t.Fatalf("created account status = %q, want %q", accountRepo.createdOrLoaded.Status, domain.AccountStatusPendingProfile)
	}
	if accountRepo.createdOrLoaded.PendingProfileNickname != "Alice" {
		t.Fatalf("created pending nickname = %q, want Alice", accountRepo.createdOrLoaded.PendingProfileNickname)
	}
	if credentialRepo.saved.PasswordHash != "hashed-password" {
		t.Fatalf("saved credential hash = %q, want hashed-password", credentialRepo.saved.PasswordHash)
	}
	if result.UserID != 7001 {
		t.Fatalf("result user id = %d, want 7001", result.UserID)
	}
	if userProfiles.input.AccountID != result.AccountID {
		t.Fatalf("user profile input account id = %d, want %d", userProfiles.input.AccountID, result.AccountID)
	}
	if userProfiles.input.Nickname != "Alice" {
		t.Fatalf("user profile input nickname = %q, want Alice", userProfiles.input.Nickname)
	}
	if accountRepo.activated.AccountID != result.AccountID {
		t.Fatalf("activated account id = %d, want %d", accountRepo.activated.AccountID, result.AccountID)
	}
	if accountRepo.activated.UserID != 7001 {
		t.Fatalf("activated user id = %d, want 7001", accountRepo.activated.UserID)
	}
	if roleRepo.assignedAccountID != result.AccountID {
		t.Fatalf("assigned role account id = %d, want %d", roleRepo.assignedAccountID, result.AccountID)
	}
	if roleRepo.assignedRole != domain.RoleUser {
		t.Fatalf("assigned role = %q, want %q", roleRepo.assignedRole, domain.RoleUser)
	}
	if roleRepo.assignOrder <= userProfiles.callOrder {
		t.Fatalf("default role assigned before user profile creation: roleOrder=%d profileOrder=%d", roleRepo.assignOrder, userProfiles.callOrder)
	}
	if accountRepo.activateOrder <= userProfiles.callOrder {
		t.Fatalf("account activated before user profile creation: activateOrder=%d profileOrder=%d", accountRepo.activateOrder, userProfiles.callOrder)
	}
	if len(outbox.messages) != 1 {
		t.Fatalf("outbox message count = %d, want 1", len(outbox.messages))
	}
	if outbox.messages[0].EventType != "auth.account.registered" {
		t.Fatalf("outbox event type = %q, want auth.account.registered", outbox.messages[0].EventType)
	}

	var payload map[string]any
	if err := json.Unmarshal(outbox.messages[0].Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal(outbox payload) error = %v", err)
	}
	if got := payload["accountId"]; got != float64(result.AccountID) {
		t.Fatalf("payload accountId = %v, want %d", got, result.AccountID)
	}
	if got := payload["userId"]; got != float64(7001) {
		t.Fatalf("payload userId = %v, want 7001", got)
	}
	if got := payload["email"]; got != "user@example.com" {
		t.Fatalf("payload email = %v, want user@example.com", got)
	}
	if _, ok := payload["occurredAt"]; !ok {
		t.Fatal("payload missing occurredAt")
	}
	if _, ok := payload["nickname"]; ok {
		t.Fatal("payload must not contain nickname")
	}
	if _, ok := payload["password"]; ok {
		t.Fatal("payload must not contain password")
	}
	if _, ok := payload["token"]; ok {
		t.Fatal("payload must not contain token")
	}
}

func TestRegisterAccountReusesPendingAccountAfterRetryAndCompletesActivation(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 5, 0, 0, time.UTC)
	trace := &callTrace{}
	accountRepo := &fakeAccountRepository{
		trace: trace,
		createOrLoadResult: domain.Account{
			ID:                     2002,
			Email:                  mustEmail(t, "user@example.com"),
			Status:                 domain.AccountStatusPendingProfile,
			PendingProfileNickname: "Retry-Alice",
			SessionVersion:         1,
			PrincipalVersion:       1,
			CreatedAt:              now.Add(-5 * time.Minute),
			UpdatedAt:              now,
		},
	}
	credentialRepo := &fakeCredentialRepository{trace: trace}
	roleRepo := &fakeRoleRepository{defaultRole: domain.RoleUser, trace: trace}
	outbox := &fakeOutboxPublisher{trace: trace}
	service := mustNewService(t, Dependencies{
		Accounts:      accountRepo,
		Credentials:   credentialRepo,
		Roles:         roleRepo,
		Sessions:      &fakeRefreshSessionStore{},
		Tokens:        &fakeTokenIssuer{},
		RefreshTokens: &fakeRefreshTokenMaterialIssuer{},
		Hasher:        &fakePasswordHasher{hash: "hashed-password-v2"},
		Blacklist:     &fakeAccessTokenBlacklist{},
		Cache:         &fakeAuthCacheStore{},
		TxRunner:      &fakeTransactionRunner{},
		Outbox:        outbox,
		UserProfiles: &fakeUserProfileClient{
			trace:  trace,
			result: ports.CreatedUserProfile{UserID: 9002},
		},
		Clock:       fixedClock{now: now},
		RateLimiter: allowRateLimiter(),
	})

	result, err := service.RegisterAccount(context.Background(), RegisterAccountCommand{
		Nickname: "Retry-Alice",
		Email:    "user@example.com",
		Password: "Password456",
	})
	if err != nil {
		t.Fatalf("RegisterAccount() error = %v", err)
	}
	if result.AccountID != 2002 {
		t.Fatalf("account id = %d, want 2002", result.AccountID)
	}
	if result.UserID != 9002 {
		t.Fatalf("user id = %d, want 9002", result.UserID)
	}
	if credentialRepo.saved.AccountID != 2002 {
		t.Fatalf("saved credential account id = %d, want 2002", credentialRepo.saved.AccountID)
	}
	if credentialRepo.saved.PasswordHash != "hashed-password-v2" {
		t.Fatalf("saved credential hash = %q, want hashed-password-v2", credentialRepo.saved.PasswordHash)
	}
	if accountRepo.activated.AccountID != 2002 {
		t.Fatalf("activated account id = %d, want 2002", accountRepo.activated.AccountID)
	}
	if len(outbox.messages) != 1 {
		t.Fatalf("outbox message count = %d, want 1", len(outbox.messages))
	}
}

func TestRegisterAccountUsesFreshActivationTimeForAccountAndOutbox(t *testing.T) {
	requestStartedAt := time.Date(2026, 7, 4, 10, 7, 0, 0, time.UTC)
	activatedAt := requestStartedAt.Add(3 * time.Minute)
	trace := &callTrace{}
	accountRepo := &fakeAccountRepository{trace: trace}
	outbox := &fakeOutboxPublisher{trace: trace}
	service := mustNewService(t, Dependencies{
		Accounts:      accountRepo,
		Credentials:   &fakeCredentialRepository{trace: trace},
		Roles:         &fakeRoleRepository{defaultRole: domain.RoleUser, trace: trace},
		Sessions:      &fakeRefreshSessionStore{},
		Tokens:        &fakeTokenIssuer{},
		RefreshTokens: &fakeRefreshTokenMaterialIssuer{},
		Hasher:        &fakePasswordHasher{hash: "hashed-password"},
		TxRunner:      &fakeTransactionRunner{},
		Outbox:        outbox,
		UserProfiles:  &fakeUserProfileClient{trace: trace, result: ports.CreatedUserProfile{UserID: 7001}},
		Clock:         &sequenceClock{times: []time.Time{requestStartedAt, activatedAt}},
		RateLimiter:   allowRateLimiter(),
	})

	result, err := service.RegisterAccount(context.Background(), RegisterAccountCommand{
		Nickname: "Alice",
		Email:    "user@example.com",
		Password: "Password123",
	})
	if err != nil {
		t.Fatalf("RegisterAccount() error = %v", err)
	}
	if result.AccountID == 0 {
		t.Fatal("result account id = 0, want non-zero")
	}
	if !accountRepo.activated.ActivatedAt.Equal(activatedAt) {
		t.Fatalf("activated at = %v, want %v", accountRepo.activated.ActivatedAt, activatedAt)
	}
	if len(outbox.messages) != 1 {
		t.Fatalf("outbox message count = %d, want 1", len(outbox.messages))
	}
	if !outbox.messages[0].OccurredAt.Equal(activatedAt) {
		t.Fatalf("outbox occurredAt = %v, want %v", outbox.messages[0].OccurredAt, activatedAt)
	}

	var payload map[string]any
	if err := json.Unmarshal(outbox.messages[0].Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal(outbox payload) error = %v", err)
	}
	if got := payload["occurredAt"]; got != activatedAt.UTC().Format(time.RFC3339) {
		t.Fatalf("payload occurredAt = %v, want %q", got, activatedAt.UTC().Format(time.RFC3339))
	}
}

func TestRegisterAccountReturnsErrorWhenUserProfileInitializationFails(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 10, 0, 0, time.UTC)
	trace := &callTrace{}
	accountRepo := &fakeAccountRepository{trace: trace}
	credentialRepo := &fakeCredentialRepository{trace: trace}
	roleRepo := &fakeRoleRepository{defaultRole: domain.RoleUser, trace: trace}
	outbox := &fakeOutboxPublisher{trace: trace}
	userProfiles := &fakeUserProfileClient{
		trace: trace,
		err:   errors.New("user unavailable"),
	}
	txRunner := &fakeTransactionRunner{}

	service := mustNewService(t, Dependencies{
		Accounts:      accountRepo,
		Credentials:   credentialRepo,
		Roles:         roleRepo,
		Sessions:      &fakeRefreshSessionStore{},
		Tokens:        &fakeTokenIssuer{},
		RefreshTokens: &fakeRefreshTokenMaterialIssuer{},
		Hasher:        &fakePasswordHasher{hash: "hashed-password"},
		Blacklist:     &fakeAccessTokenBlacklist{},
		Cache:         &fakeAuthCacheStore{},
		TxRunner:      txRunner,
		Outbox:        outbox,
		UserProfiles:  userProfiles,
		Clock:         fixedClock{now: now},
		RateLimiter:   allowRateLimiter(),
	})

	_, err := service.RegisterAccount(context.Background(), RegisterAccountCommand{
		Nickname: "Alice",
		Email:    "user@example.com",
		Password: "Password123",
	})
	if err == nil {
		t.Fatal("RegisterAccount() error = nil, want failure")
	}

	if txRunner.calledCount != 1 {
		t.Fatalf("transaction count = %d, want 1", txRunner.calledCount)
	}
	if accountRepo.activateOrder != 0 {
		t.Fatalf("activate order = %d, want 0", accountRepo.activateOrder)
	}
	if roleRepo.assignOrder != 0 {
		t.Fatalf("role assign order = %d, want 0", roleRepo.assignOrder)
	}
	if len(outbox.messages) != 0 {
		t.Fatalf("outbox message count = %d, want 0", len(outbox.messages))
	}
}

func TestRegisterAccountRejectsZeroUserIDFromUserProfileContract(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 20, 0, 0, time.UTC)
	trace := &callTrace{}
	accountRepo := &fakeAccountRepository{trace: trace}
	roleRepo := &fakeRoleRepository{defaultRole: domain.RoleUser, trace: trace}
	outbox := &fakeOutboxPublisher{trace: trace}
	txRunner := &fakeTransactionRunner{}
	service := mustNewService(t, Dependencies{
		Accounts:      accountRepo,
		Credentials:   &fakeCredentialRepository{trace: trace},
		Roles:         roleRepo,
		Sessions:      &fakeRefreshSessionStore{},
		Tokens:        &fakeTokenIssuer{},
		RefreshTokens: &fakeRefreshTokenMaterialIssuer{},
		Hasher:        &fakePasswordHasher{hash: "hashed-password"},
		Blacklist:     &fakeAccessTokenBlacklist{},
		Cache:         &fakeAuthCacheStore{},
		TxRunner:      txRunner,
		Outbox:        outbox,
		UserProfiles:  &fakeUserProfileClient{trace: trace, result: ports.CreatedUserProfile{}},
		Clock:         fixedClock{now: now},
		RateLimiter:   allowRateLimiter(),
	})

	_, err := service.RegisterAccount(context.Background(), RegisterAccountCommand{
		Nickname: "Alice",
		Email:    "user@example.com",
		Password: "Password123",
	})
	if err == nil {
		t.Fatal("RegisterAccount() error = nil, want contract violation")
	}
	if txRunner.calledCount != 1 {
		t.Fatalf("transaction count = %d, want 1", txRunner.calledCount)
	}
	if accountRepo.activateOrder != 0 {
		t.Fatalf("activate order = %d, want 0", accountRepo.activateOrder)
	}
	if roleRepo.assignOrder != 0 {
		t.Fatalf("role assign order = %d, want 0", roleRepo.assignOrder)
	}
	if len(outbox.messages) != 0 {
		t.Fatalf("outbox message count = %d, want 0", len(outbox.messages))
	}
}

func mustEmail(t *testing.T, raw string) domain.Email {
	t.Helper()

	email, err := domain.NewEmail(raw)
	if err != nil {
		t.Fatalf("domain.NewEmail(%q) error = %v", raw, err)
	}
	return email
}

func mustNewService(t *testing.T, deps Dependencies) *Service {
	t.Helper()

	service, err := NewService(deps)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func allowRateLimiter() *fakeRateLimiter {
	return &fakeRateLimiter{result: ports.RateLimitResult{Outcome: ports.RateLimitOutcomeAllow}}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceClock struct {
	times []time.Time
	index int
}

func (c *sequenceClock) Now() time.Time {
	if len(c.times) == 0 {
		return time.Time{}
	}
	if c.index >= len(c.times) {
		return c.times[len(c.times)-1]
	}
	now := c.times[c.index]
	c.index++
	return now
}

type callTrace struct {
	next int
}

func (c *callTrace) Record() int {
	c.next++
	return c.next
}

type fakeTransactionRunner struct {
	calledCount int
}

func (f *fakeTransactionRunner) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	f.calledCount++
	return fn(ctx)
}

type fakeAccountRepository struct {
	trace              *callTrace
	found              domain.Account
	createOrLoadErr    error
	createOrLoadInput  ports.CreateOrLoadPendingAccountInput
	createOrLoadResult domain.Account
	createdOrLoaded    domain.Account
	activateErr        error
	activated          ports.ActivateAccountInput
	createOrder        int
	activateOrder      int
}

func (f *fakeAccountRepository) FindByEmail(ctx context.Context, email domain.Email) (domain.Account, error) {
	if f.found.ID == 0 {
		return domain.Account{}, domain.ErrAccountNotFound
	}
	return f.found, nil
}

func (f *fakeAccountRepository) CreateOrLoadPendingForRegister(ctx context.Context, input ports.CreateOrLoadPendingAccountInput) (domain.Account, error) {
	f.createOrLoadInput = input
	if f.trace != nil {
		f.createOrder = f.trace.Record()
	}
	if f.createOrLoadErr != nil {
		return domain.Account{}, f.createOrLoadErr
	}
	account := f.createOrLoadResult
	if account.ID == 0 {
		account = domain.NewPendingAccount(input.Email, input.Nickname, input.Now)
		account.ID = 1001
	} else {
		account.Email = input.Email
		account.PendingProfileNickname = input.Nickname
		account.UpdatedAt = input.Now
	}
	f.createdOrLoaded = account
	return account, nil
}

func (f *fakeAccountRepository) Activate(ctx context.Context, input ports.ActivateAccountInput) (domain.Account, error) {
	if f.trace != nil {
		f.activateOrder = f.trace.Record()
	}
	if f.activateErr != nil {
		return domain.Account{}, f.activateErr
	}
	f.activated = input
	return domain.Account{
		ID:                     input.AccountID,
		UserID:                 input.UserID,
		Email:                  f.createdOrLoaded.Email,
		Status:                 domain.AccountStatusActive,
		SessionVersion:         f.createdOrLoaded.SessionVersion,
		PrincipalVersion:       f.createdOrLoaded.PrincipalVersion,
		PendingProfileNickname: "",
		CreatedAt:              f.createdOrLoaded.CreatedAt,
		UpdatedAt:              input.ActivatedAt,
	}, nil
}

type fakeCredentialRepository struct {
	trace      *callTrace
	saved      domain.Credential
	stored     domain.Credential
	findCalled bool
}

func (f *fakeCredentialRepository) SaveForPendingAccount(ctx context.Context, credential domain.Credential) error {
	if f.trace != nil {
		f.trace.Record()
	}
	f.saved = credential
	return nil
}

func (f *fakeCredentialRepository) FindByAccountID(ctx context.Context, accountID domain.AccountID) (domain.Credential, error) {
	f.findCalled = true
	if f.stored.AccountID == 0 {
		return domain.Credential{}, domain.ErrCredentialNotFound
	}
	return f.stored, nil
}

type fakeRoleRepository struct {
	trace             *callTrace
	defaultRole       domain.RoleName
	assignedAccountID domain.AccountID
	assignedRole      domain.RoleName
	assignOrder       int
	roles             []domain.RoleName
}

func (f *fakeRoleRepository) DefaultRole(ctx context.Context) (domain.RoleName, error) {
	return f.defaultRole, nil
}

func (f *fakeRoleRepository) Assign(ctx context.Context, accountID domain.AccountID, role domain.RoleName) error {
	if f.trace != nil {
		f.assignOrder = f.trace.Record()
	}
	f.assignedAccountID = accountID
	f.assignedRole = role
	return nil
}

func (f *fakeRoleRepository) ListByAccountID(ctx context.Context, accountID domain.AccountID) ([]domain.RoleName, error) {
	return append([]domain.RoleName(nil), f.roles...), nil
}

type fakeRefreshSessionStore struct {
	trace       *callTrace
	createErr   error
	createInput ports.CreateRefreshSessionInput
	createOrder int
}

func (f *fakeRefreshSessionStore) Create(ctx context.Context, input ports.CreateRefreshSessionInput) error {
	f.createInput = input
	if f.trace != nil {
		f.createOrder = f.trace.Record()
	} else {
		f.createOrder++
	}
	if f.createErr != nil {
		return f.createErr
	}
	return nil
}

type fakeRefreshTokenMaterialIssuer struct {
	trace      *callTrace
	request    ports.GenerateRefreshTokenMaterialInput
	material   ports.GeneratedRefreshTokenMaterial
	err        error
	issueOrder int
}

func (f *fakeRefreshTokenMaterialIssuer) GenerateLoginMaterial(ctx context.Context, input ports.GenerateRefreshTokenMaterialInput) (ports.GeneratedRefreshTokenMaterial, error) {
	if f.trace != nil {
		f.issueOrder = f.trace.Record()
	} else {
		f.issueOrder++
	}
	f.request = input
	if f.err != nil {
		return ports.GeneratedRefreshTokenMaterial{}, f.err
	}
	if f.material == (ports.GeneratedRefreshTokenMaterial{}) {
		f.material = ports.GeneratedRefreshTokenMaterial{
			SessionID: "session-default",
			TokenID:   "token-id-default",
			Plaintext: "refresh-token-default",
			TokenHash: "token-hash-default",
			ExpiresAt: input.IssuedAt.Add(30 * 24 * time.Hour),
		}
	}
	return f.material, nil
}

type fakeTokenIssuer struct {
	trace      *callTrace
	request    ports.IssueAccessTokenRequest
	token      ports.IssuedAccessToken
	err        error
	issueOrder int
}

func (f *fakeTokenIssuer) IssueLoginAccessToken(ctx context.Context, request ports.IssueAccessTokenRequest) (ports.IssuedAccessToken, error) {
	if f.trace != nil {
		f.issueOrder = f.trace.Record()
	} else {
		f.issueOrder++
	}
	f.request = request
	if f.err != nil {
		return ports.IssuedAccessToken{}, f.err
	}
	return f.token, nil
}

type fakePasswordHasher struct {
	hash         string
	verifyOK     bool
	verifyCalled bool
}

func (f *fakePasswordHasher) HashPassword(ctx context.Context, password string) (string, error) {
	return f.hash, nil
}

func (f *fakePasswordHasher) VerifyPassword(ctx context.Context, password string, passwordHash string) (bool, error) {
	f.verifyCalled = true
	return f.verifyOK, nil
}

type fakeAccessTokenBlacklist struct{}

func (f *fakeAccessTokenBlacklist) Blacklist(ctx context.Context, tokenID string, expiresAt time.Time) error {
	return nil
}

type fakeAuthCacheStore struct{}

func (f *fakeAuthCacheStore) StorePrincipal(ctx context.Context, principal ports.PrincipalSnapshot) error {
	return nil
}

func (f *fakeAuthCacheStore) MarkSessionRevoked(ctx context.Context, sessionID string, expiresAt time.Time) error {
	return nil
}

func (f *fakeAuthCacheStore) SetSessionVersion(ctx context.Context, accountID domain.AccountID, version int64) error {
	return nil
}

func (f *fakeAuthCacheStore) SetPrincipalVersion(ctx context.Context, accountID domain.AccountID, version int64) error {
	return nil
}

type fakeOutboxPublisher struct {
	trace    *callTrace
	messages []ports.OutboxMessage
}

func (f *fakeOutboxPublisher) Publish(ctx context.Context, message ports.OutboxMessage) error {
	if f.trace != nil {
		f.trace.Record()
	}
	f.messages = append(f.messages, message)
	return nil
}

type fakeUserProfileClient struct {
	trace     *callTrace
	input     ports.CreateProfileForAccountInput
	result    ports.CreatedUserProfile
	err       error
	callOrder int
}

func (f *fakeUserProfileClient) CreateProfileForAccount(ctx context.Context, input ports.CreateProfileForAccountInput) (ports.CreatedUserProfile, error) {
	if f.trace != nil {
		f.callOrder = f.trace.Record()
	}
	f.input = input
	if f.err != nil {
		return ports.CreatedUserProfile{}, f.err
	}
	return f.result, nil
}

type fakeRateLimiter struct {
	result ports.RateLimitResult
	err    error
}

func (f *fakeRateLimiter) Check(ctx context.Context, operation domain.SecurityOperation, key string) (ports.RateLimitResult, error) {
	if f.err != nil {
		return ports.RateLimitResult{}, f.err
	}
	return f.result, nil
}
