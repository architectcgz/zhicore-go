package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func TestMarkNotificationReadRejectsInvalidPublicIDBeforeRepositoryLookup(t *testing.T) {
	deps := newReadTestDeps()
	deps.ids.decodeErr = errors.New("bad checksum")
	service := mustNewService(t, deps.dependencies())

	_, err := service.MarkNotificationRead(context.Background(), MarkNotificationReadCommand{
		Actor:          Actor{UserID: 42},
		NotificationID: "bad-id",
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("MarkNotificationRead() error = %v, want %v", err, ErrInvalidRequest)
	}
	if deps.commands.markReadCalls != 0 {
		t.Fatalf("repository mark read calls = %d, want 0", deps.commands.markReadCalls)
	}
}

func TestMarkNotificationReadUsesRecipientScopeAndInvalidatesUnreadCache(t *testing.T) {
	deps := newReadTestDeps()
	now := time.Date(2026, 7, 6, 15, 0, 0, 0, time.UTC)
	deps.clock.now = now
	deps.ids.decoded = 1001
	deps.commands.markReadResult = ports.MarkReadResult{NotificationID: 1001, PublicID: "n1abc", Changed: true, ReadAt: now}
	service := mustNewService(t, deps.dependencies())

	result, err := service.MarkNotificationRead(context.Background(), MarkNotificationReadCommand{
		Actor:          Actor{UserID: 42},
		NotificationID: "n1abc",
	})
	if err != nil {
		t.Fatalf("MarkNotificationRead() error = %v", err)
	}
	if result.NotificationID != "n1abc" || !result.Read || result.ReadAt != now {
		t.Fatalf("result = %#v", result)
	}
	if deps.commands.lastMarkRead.NotificationID != 1001 || deps.commands.lastMarkRead.RecipientID != 42 {
		t.Fatalf("mark read input = %#v, want notification 1001 recipient 42", deps.commands.lastMarkRead)
	}
	if deps.cache.deleted["notification:42:unread"] != 1 || deps.cache.deleted["notification:42:aggregation"] != 1 {
		t.Fatalf("deleted cache keys = %#v", deps.cache.deleted)
	}
}

func TestMarkNotificationReadKeepsRepeatReadIdempotent(t *testing.T) {
	deps := newReadTestDeps()
	readAt := time.Date(2026, 7, 6, 14, 30, 0, 0, time.UTC)
	deps.ids.decoded = 1001
	deps.commands.markReadResult = ports.MarkReadResult{NotificationID: 1001, PublicID: "n1abc", Changed: false, ReadAt: readAt}
	service := mustNewService(t, deps.dependencies())

	result, err := service.MarkNotificationRead(context.Background(), MarkNotificationReadCommand{
		Actor:          Actor{UserID: 42},
		NotificationID: "n1abc",
	})
	if err != nil {
		t.Fatalf("MarkNotificationRead() error = %v", err)
	}
	if result.Changed || result.ReadAt != readAt {
		t.Fatalf("repeat read result = %#v", result)
	}
}

func TestMarkAllNotificationsReadInvalidatesUserNotificationKeys(t *testing.T) {
	deps := newReadTestDeps()
	now := time.Date(2026, 7, 6, 15, 10, 0, 0, time.UTC)
	deps.clock.now = now
	deps.commands.markAllReadResult = ports.MarkAllReadResult{AffectedCount: 7, ReadAt: now}
	service := mustNewService(t, deps.dependencies())

	result, err := service.MarkAllNotificationsRead(context.Background(), MarkAllNotificationsReadCommand{Actor: Actor{UserID: 42}})
	if err != nil {
		t.Fatalf("MarkAllNotificationsRead() error = %v", err)
	}
	if !result.ReadAll || result.AffectedCount != 7 || result.ReadAt != now {
		t.Fatalf("result = %#v", result)
	}
	if deps.commands.lastMarkAllRead.RecipientID != 42 {
		t.Fatalf("mark all input = %#v, want recipient 42", deps.commands.lastMarkAllRead)
	}
	if deps.cache.deleted["notification:42:unread"] != 1 || deps.cache.deleted["notification:42:aggregation"] != 1 {
		t.Fatalf("deleted cache keys = %#v", deps.cache.deleted)
	}
}

type readTestDeps struct {
	commands *fakeCommandRepository
	queries  *fakeQueryRepository
	cache    *fakeUnreadCache
	ids      *fakePublicIDCodec
	clock    *fakeClock
}

func newReadTestDeps() readTestDeps {
	return readTestDeps{
		commands: &fakeCommandRepository{},
		queries:  &fakeQueryRepository{},
		cache:    &fakeUnreadCache{deleted: map[string]int{}},
		ids:      &fakePublicIDCodec{},
		clock:    &fakeClock{now: time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)},
	}
}

func (d readTestDeps) dependencies() Dependencies {
	return Dependencies{
		Commands: d.commands,
		Queries:  d.queries,
		Unread:   d.cache,
		IDs:      d.ids,
		Clock:    d.clock,
	}
}

func mustNewService(t *testing.T, deps Dependencies) *Service {
	t.Helper()
	service, err := NewService(deps)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

type fakeCommandRepository struct {
	markReadCalls     int
	lastMarkRead      ports.MarkReadInput
	markReadResult    ports.MarkReadResult
	markReadErr       error
	lastMarkAllRead   ports.MarkAllReadInput
	markAllReadResult ports.MarkAllReadResult
	markAllReadErr    error
}

func (f *fakeCommandRepository) MarkRead(ctx context.Context, input ports.MarkReadInput) (ports.MarkReadResult, error) {
	f.markReadCalls++
	f.lastMarkRead = input
	return f.markReadResult, f.markReadErr
}

func (f *fakeCommandRepository) MarkAllRead(ctx context.Context, input ports.MarkAllReadInput) (ports.MarkAllReadResult, error) {
	f.lastMarkAllRead = input
	return f.markAllReadResult, f.markAllReadErr
}

type fakeQueryRepository struct{}

func (f *fakeQueryRepository) GetUnreadCount(ctx context.Context, recipientID int64) (int64, error) {
	return 0, nil
}

func (f *fakeQueryRepository) GetUnreadBreakdown(ctx context.Context, recipientID int64) (ports.UnreadBreakdown, error) {
	return ports.UnreadBreakdown{}, nil
}

func (f *fakeQueryRepository) ListAggregated(ctx context.Context, query ports.ListAggregatedQuery) (ports.AggregatedNotificationPage, error) {
	return ports.AggregatedNotificationPage{}, nil
}

type fakeUnreadCache struct {
	deleted map[string]int
}

func (f *fakeUnreadCache) GetUnreadCount(ctx context.Context, userID int64) (int64, bool, error) {
	return 0, false, nil
}

func (f *fakeUnreadCache) SetUnreadCount(ctx context.Context, userID int64, count int64) error {
	return nil
}

func (f *fakeUnreadCache) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		f.deleted[key]++
	}
	return nil
}

type fakePublicIDCodec struct {
	decoded   uint64
	decodeErr error
}

func (f *fakePublicIDCodec) Encode(id uint64) (string, error) {
	return "", nil
}

func (f *fakePublicIDCodec) Decode(publicID string) (uint64, error) {
	if f.decodeErr != nil {
		return 0, f.decodeErr
	}
	return f.decoded, nil
}

type fakeClock struct {
	now time.Time
}

func (f *fakeClock) Now() time.Time {
	return f.now
}
