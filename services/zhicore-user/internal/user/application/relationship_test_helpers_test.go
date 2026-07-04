package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/ports"
)

func mustNewRelationshipService(t *testing.T, store *fakeProfileStore, relationships *fakeRelationshipStore, outbox *fakeOutboxPublisher, now time.Time) *Service {
	t.Helper()
	return mustNewService(t, Dependencies{
		Profiles:      store,
		Queries:       store,
		Relationships: relationships,
		Files:         &fakeFileReferenceClient{},
		IDs:           &fakePublicIDGenerator{},
		Outbox:        outbox,
		TxRunner:      &fakeTransactionRunner{},
		Clock:         fixedClock{now: now},
		Cache:         &fakeCacheStore{},
	})
}

func seedRelationshipProfile(t *testing.T, store *fakeProfileStore, userID domain.UserID, publicID domain.PublicID, status domain.UserStatus) domain.Profile {
	t.Helper()
	return store.seedProfile(t, domain.ProfileSeed{
		UserID:                 userID,
		PublicID:               publicID,
		AccountID:              domain.AccountID(userID + 1000),
		Nickname:               "User" + string(publicID)[len("user_pub_"):],
		StrangerMessageAllowed: true,
		Status:                 status,
		CreatedAt:              time.Date(2026, 7, 4, 13, 0, 0, 0, time.UTC),
		UpdatedAt:              time.Date(2026, 7, 4, 13, 0, 0, 0, time.UTC),
	})
}

func assertEventTypes(t *testing.T, messages []ports.OutboxMessage, want []string) {
	t.Helper()
	if len(messages) != len(want) {
		t.Fatalf("outbox event count = %d, want %d: %#v", len(messages), len(want), messages)
	}
	for i := range want {
		if messages[i].EventType != want[i] {
			t.Fatalf("outbox event[%d] = %q, want %q", i, messages[i].EventType, want[i])
		}
	}
}

func assertEventPayloadField(t *testing.T, message ports.OutboxMessage, field string, want string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}
	if payload[field] != want {
		t.Fatalf("payload[%q] = %#v, want %q; payload=%#v", field, payload[field], want, payload)
	}
}

func assertFollowStats(t *testing.T, store *fakeRelationshipStore, userID domain.UserID, followers, following int64) {
	t.Helper()
	got := store.stats[userID]
	if got.FollowersCount != followers || got.FollowingCount != following {
		t.Fatalf("stats[%d] = %#v, want followers=%d following=%d", userID, got, followers, following)
	}
}

type fakeRelationshipStore struct {
	nextID  int64
	follows map[domain.UserPair]ports.RelationshipRecord
	blocks  map[domain.UserPair]ports.RelationshipRecord
	stats   map[domain.UserID]ports.FollowStats
}

func newFakeRelationshipStore() *fakeRelationshipStore {
	return &fakeRelationshipStore{
		nextID:  1,
		follows: map[domain.UserPair]ports.RelationshipRecord{},
		blocks:  map[domain.UserPair]ports.RelationshipRecord{},
		stats:   map[domain.UserID]ports.FollowStats{},
	}
}

func (s *fakeRelationshipStore) seedFollow(followerID, followingID domain.UserID, createdAt time.Time) {
	_, _ = s.InsertFollow(context.Background(), followerID, followingID, createdAt)
}

func (s *fakeRelationshipStore) seedBlock(blockerID, blockedID domain.UserID, createdAt time.Time) {
	_, _ = s.InsertBlock(context.Background(), blockerID, blockedID, "", createdAt)
}

func (s *fakeRelationshipStore) hasFollow(followerID, followingID domain.UserID) bool {
	_, ok := s.follows[domain.UserPair{ActorID: followerID, TargetID: followingID}]
	return ok
}

func (s *fakeRelationshipStore) hasBlock(blockerID, blockedID domain.UserID) bool {
	_, ok := s.blocks[domain.UserPair{ActorID: blockerID, TargetID: blockedID}]
	return ok
}

func (s *fakeRelationshipStore) InsertFollow(_ context.Context, followerID, followingID domain.UserID, now time.Time) (bool, error) {
	pair := domain.UserPair{ActorID: followerID, TargetID: followingID}
	if _, ok := s.follows[pair]; ok {
		return false, nil
	}
	s.follows[pair] = ports.RelationshipRecord{ID: s.nextID, ActorID: followerID, TargetID: followingID, CreatedAt: now}
	s.nextID++
	s.adjustStats(followingID, 1, 0)
	s.adjustStats(followerID, 0, 1)
	return true, nil
}

func (s *fakeRelationshipStore) DeleteFollow(_ context.Context, followerID, followingID domain.UserID) (bool, error) {
	pair := domain.UserPair{ActorID: followerID, TargetID: followingID}
	if _, ok := s.follows[pair]; !ok {
		return false, nil
	}
	delete(s.follows, pair)
	s.adjustStats(followingID, -1, 0)
	s.adjustStats(followerID, 0, -1)
	return true, nil
}

func (s *fakeRelationshipStore) InsertBlock(_ context.Context, blockerID, blockedID domain.UserID, reason string, now time.Time) (bool, error) {
	pair := domain.UserPair{ActorID: blockerID, TargetID: blockedID}
	if _, ok := s.blocks[pair]; ok {
		return false, nil
	}
	s.blocks[pair] = ports.RelationshipRecord{ID: s.nextID, ActorID: blockerID, TargetID: blockedID, Reason: reason, CreatedAt: now}
	s.nextID++
	return true, nil
}

func (s *fakeRelationshipStore) DeleteBlock(_ context.Context, blockerID, blockedID domain.UserID) (bool, error) {
	pair := domain.UserPair{ActorID: blockerID, TargetID: blockedID}
	if _, ok := s.blocks[pair]; !ok {
		return false, nil
	}
	delete(s.blocks, pair)
	return true, nil
}

func (s *fakeRelationshipStore) ListBlocked(ctx context.Context, blockerID domain.UserID, cursor string, limit int) (ports.RelationshipPage, error) {
	return s.list(ctx, s.blocks, blockerID, cursor, limit, true)
}

func (s *fakeRelationshipStore) ListFollowers(ctx context.Context, targetID domain.UserID, cursor string, limit int) (ports.RelationshipPage, error) {
	return s.list(ctx, s.follows, targetID, cursor, limit, false)
}

func (s *fakeRelationshipStore) ListFollowing(ctx context.Context, targetID domain.UserID, cursor string, limit int) (ports.RelationshipPage, error) {
	return s.list(ctx, s.follows, targetID, cursor, limit, true)
}

func (s *fakeRelationshipStore) BatchCheckBlocked(_ context.Context, pairs []domain.UserPair) (map[domain.UserPair]bool, error) {
	result := make(map[domain.UserPair]bool, len(pairs))
	for _, pair := range pairs {
		_, ok := s.blocks[pair]
		result[pair] = ok
	}
	return result, nil
}

func (s *fakeRelationshipStore) CheckFollowing(_ context.Context, followerID, followingID domain.UserID) (bool, error) {
	_, ok := s.follows[domain.UserPair{ActorID: followerID, TargetID: followingID}]
	return ok, nil
}

func (s *fakeRelationshipStore) list(_ context.Context, records map[domain.UserPair]ports.RelationshipRecord, ownerID domain.UserID, cursor string, limit int, ownerIsActor bool) (ports.RelationshipPage, error) {
	afterID, err := domain.DecodeRelationshipCursor(cursor)
	if err != nil {
		return ports.RelationshipPage{}, err
	}
	limit = domain.NormalizeRelationshipLimit(limit)
	var rows []ports.RelationshipRecord
	for _, record := range records {
		if ownerIsActor && record.ActorID != ownerID {
			continue
		}
		if !ownerIsActor && record.TargetID != ownerID {
			continue
		}
		if afterID > 0 && record.ID >= afterID {
			continue
		}
		rows = append(rows, record)
	}
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].ID > rows[i].ID {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	return ports.RelationshipPage{Records: rows, HasMore: hasMore}, nil
}

func (s *fakeRelationshipStore) adjustStats(userID domain.UserID, followersDelta, followingDelta int64) {
	stats := s.stats[userID]
	stats.FollowersCount += followersDelta
	stats.FollowingCount += followingDelta
	if stats.FollowersCount < 0 {
		stats.FollowersCount = 0
	}
	if stats.FollowingCount < 0 {
		stats.FollowingCount = 0
	}
	s.stats[userID] = stats
}
