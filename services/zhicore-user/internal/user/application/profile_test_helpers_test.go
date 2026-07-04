package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/ports"
)

func mustNewService(t *testing.T, deps Dependencies) *Service {
	t.Helper()
	if deps.CacheFailures == nil {
		deps.CacheFailures = &fakeCacheFailureRecorder{}
	}
	service, err := NewService(deps)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

var errAbortedTransaction = errors.New("transaction aborted after unique violation")

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type callTrace struct{ calls []string }

func (t *callTrace) add(call string) { t.calls = append(t.calls, call) }

type fakeProfileStore struct {
	trace                                                         *callTrace
	nextUserID                                                    int64
	byAccount                                                     map[domain.AccountID]domain.Profile
	byUserID                                                      map[domain.UserID]domain.Profile
	byPublicID                                                    map[domain.PublicID]domain.Profile
	byNickname                                                    map[string]domain.UserID
	createCalls, createOrGetCalls, updateCalls, updatePublicCalls int
	hideAccountLookup                                             map[domain.AccountID]int
	abortReloadOnConflict, conflictReloadPoisoned                 map[domain.AccountID]bool
	markDeletedErrors                                             map[domain.UserID]error
	publicUpdateVersionDelta                                      int64
	lastPublicUpdateInput                                         domain.Profile
}

type fakeCacheStore struct {
	err error
}

type fakeCacheFailure struct {
	operation string
	keys      []string
	err       error
}

type fakeCacheFailureRecorder struct {
	failures []fakeCacheFailure
}

type fakeFileReferenceClient struct {
	trace *callTrace
	err   error
}

type fakePublicIDGenerator struct {
	ids   []domain.PublicID
	calls int
}

type fakeOutboxPublisher struct{ messages []ports.OutboxMessage }

type fakeTransactionRunner struct {
	trace       *callTrace
	calledCount int
}

func newFakeProfileStore() *fakeProfileStore {
	return &fakeProfileStore{
		nextUserID:             1,
		byAccount:              map[domain.AccountID]domain.Profile{},
		byUserID:               map[domain.UserID]domain.Profile{},
		byPublicID:             map[domain.PublicID]domain.Profile{},
		byNickname:             map[string]domain.UserID{},
		hideAccountLookup:      map[domain.AccountID]int{},
		abortReloadOnConflict:  map[domain.AccountID]bool{},
		conflictReloadPoisoned: map[domain.AccountID]bool{},
		markDeletedErrors:      map[domain.UserID]error{},
	}
}

func (s *fakeProfileStore) traceCall(call string) {
	if s.trace != nil {
		s.trace.add(call)
	}
}

func (s *fakeProfileStore) hideAccountLookupOnce(accountID domain.AccountID) {
	s.hideAccountLookup[accountID] = 1
}

func (s *fakeProfileStore) abortReloadAfterConflict(accountID domain.AccountID) {
	s.abortReloadOnConflict[accountID] = true
}

func (s *fakeProfileStore) mustProfileByAccount(accountID domain.AccountID) (domain.Profile, error) {
	profile, ok := s.byAccount[accountID]
	if !ok {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	return profile, nil
}

func (s *fakeProfileStore) seedProfile(t *testing.T, seed domain.ProfileSeed) domain.Profile {
	t.Helper()
	profile, err := domain.NewProfile(seed)
	if err != nil {
		t.Fatalf("domain.NewProfile() error = %v", err)
	}
	s.store(profile)
	if int64(profile.UserID) >= s.nextUserID {
		s.nextUserID = int64(profile.UserID) + 1
	}
	return profile
}

func (s *fakeProfileStore) FindByAccountID(ctx context.Context, accountID domain.AccountID) (domain.Profile, error) {
	s.traceCall("repo.find_by_account")
	if s.conflictReloadPoisoned[accountID] {
		delete(s.conflictReloadPoisoned, accountID)
		return domain.Profile{}, errAbortedTransaction
	}
	if misses := s.hideAccountLookup[accountID]; misses > 0 {
		s.hideAccountLookup[accountID] = misses - 1
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	return s.mustProfileByAccount(accountID)
}

func (s *fakeProfileStore) Create(ctx context.Context, profile domain.Profile) (domain.Profile, error) {
	s.traceCall("repo.create")
	s.createCalls++
	if _, exists := s.byAccount[profile.AccountID]; exists {
		if s.abortReloadOnConflict[profile.AccountID] {
			s.conflictReloadPoisoned[profile.AccountID] = true
		}
		return domain.Profile{}, domain.ErrAccountAlreadyExists
	}
	if userID, exists := s.byNickname[profile.Nickname]; exists && userID != profile.UserID {
		return domain.Profile{}, domain.ErrNicknameTaken
	}
	if profile.UserID == 0 {
		profile.UserID = domain.UserID(s.nextUserID)
		s.nextUserID++
	}
	s.store(profile)
	return profile, nil
}

func (s *fakeProfileStore) CreateOrGetByAccountID(ctx context.Context, profile domain.Profile) (domain.Profile, bool, error) {
	s.traceCall("repo.create_or_get_by_account")
	s.createOrGetCalls++
	if existing, ok := s.byAccount[profile.AccountID]; ok {
		return existing, false, nil
	}
	created, err := s.Create(ctx, profile)
	return created, err == nil, err
}

func (s *fakeProfileStore) Update(ctx context.Context, profile domain.Profile) (domain.Profile, error) {
	s.traceCall("repo.update")
	s.updateCalls++
	current, ok := s.byUserID[profile.UserID]
	if !ok {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	if userID, exists := s.byNickname[profile.Nickname]; exists && userID != profile.UserID {
		return domain.Profile{}, domain.ErrNicknameTaken
	}
	delete(s.byNickname, current.Nickname)
	s.store(profile)
	return profile, nil
}

func (s *fakeProfileStore) UpdatePublicProfile(ctx context.Context, profile domain.Profile) (domain.Profile, error) {
	s.traceCall("repo.update_public")
	s.updatePublicCalls++
	current, ok := s.byUserID[profile.UserID]
	if !ok {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	if userID, exists := s.byNickname[profile.Nickname]; exists && userID != profile.UserID {
		return domain.Profile{}, domain.ErrNicknameTaken
	}
	s.lastPublicUpdateInput = profile
	profile.ProfileVersion = current.ProfileVersion + 1 + s.publicUpdateVersionDelta
	delete(s.byNickname, current.Nickname)
	s.store(profile)
	return profile, nil
}

func (s *fakeProfileStore) GetByUserID(ctx context.Context, userID domain.UserID) (domain.Profile, error) {
	s.traceCall("query.get_by_user_id")
	profile, ok := s.byUserID[userID]
	if !ok {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	return profile, nil
}

func (s *fakeProfileStore) GetByPublicID(ctx context.Context, publicID domain.PublicID) (domain.Profile, error) {
	s.traceCall("query.get_by_public_id")
	profile, ok := s.byPublicID[publicID]
	if !ok {
		return domain.Profile{}, domain.ErrProfileNotFound
	}
	return profile, nil
}

func (s *fakeProfileStore) store(profile domain.Profile) {
	s.byAccount[profile.AccountID] = profile
	s.byUserID[profile.UserID] = profile
	s.byPublicID[profile.PublicID] = profile
	s.byNickname[profile.Nickname] = profile.UserID
}

func (s *fakeProfileStore) DeactivateByAccountID(ctx context.Context, accountID domain.AccountID, now time.Time) (domain.Profile, bool, error) {
	s.traceCall("repo.deactivate_by_account")
	profile, err := s.mustProfileByAccount(accountID)
	if err != nil {
		return domain.Profile{}, false, err
	}
	updated, changed, err := profile.Deactivate(now)
	if err != nil || !changed {
		return updated, changed, err
	}
	s.store(updated)
	return updated, true, nil
}

func (s *fakeProfileStore) MarkDeleted(ctx context.Context, userID, operatorID domain.UserID, reason string, now time.Time) (domain.Profile, bool, error) {
	s.traceCall("repo.mark_deleted")
	if err := s.markDeletedErrors[userID]; err != nil {
		return domain.Profile{}, false, err
	}
	profile, ok := s.byUserID[userID]
	if !ok {
		return domain.Profile{}, false, domain.ErrProfileNotFound
	}
	updated, changed := profile.MarkDeleted(operatorID, reason, now)
	if changed {
		s.store(updated)
	}
	return updated, changed, nil
}

func (s *fakeProfileStore) RestoreDeleted(ctx context.Context, userID, operatorID domain.UserID, reason string, now time.Time) (domain.Profile, bool, error) {
	s.traceCall("repo.restore_deleted")
	profile, ok := s.byUserID[userID]
	if !ok {
		return domain.Profile{}, false, domain.ErrProfileNotFound
	}
	updated, changed, err := profile.RestoreDeleted(operatorID, reason, now)
	if err != nil || !changed {
		return updated, changed, err
	}
	s.store(updated)
	return updated, true, nil
}

func (c *fakeCacheStore) Delete(ctx context.Context, keys ...string) error { return c.err }

func (r *fakeCacheFailureRecorder) RecordCacheDeleteFailure(ctx context.Context, operation string, keys []string, err error) {
	copiedKeys := append([]string(nil), keys...)
	r.failures = append(r.failures, fakeCacheFailure{operation: operation, keys: copiedKeys, err: err})
}

func (c *fakeFileReferenceClient) EnsureAvatarReferenced(ctx context.Context, fileID string) error {
	if c.trace != nil {
		c.trace.add("file.validate")
	}
	return c.err
}

func (g *fakePublicIDGenerator) Generate(ctx context.Context) (domain.PublicID, error) {
	g.calls++
	if len(g.ids) == 0 {
		return "", errors.New("no public ids configured")
	}
	id := g.ids[0]
	g.ids = g.ids[1:]
	return id, nil
}

func (p *fakeOutboxPublisher) Publish(ctx context.Context, message ports.OutboxMessage) error {
	p.messages = append(p.messages, message)
	return nil
}

func (r *fakeTransactionRunner) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	r.calledCount++
	if r.trace != nil {
		r.trace.add("tx.start")
	}
	return fn(ctx)
}

func optionalString(value string) *string { return &value }

func optionalBool(value bool) *bool { return &value }
