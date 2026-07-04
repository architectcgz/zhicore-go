package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
)

func TestCreateProfileSupportsIdempotentCreationAndQueries(t *testing.T) {
	now := time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC)

	t.Run("creates profile and publishes created event", func(t *testing.T) {
		store := newFakeProfileStore()
		outbox := &fakeOutboxPublisher{}
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{ids: []domain.PublicID{"user_pub_1"}},
			Outbox: outbox, TxRunner: &fakeTransactionRunner{}, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})

		profile, err := service.CreateProfileForAccount(context.Background(), CreateProfileForAccountCommand{
			AccountID: 1001,
			Username:  " Alice ",
		})
		if err != nil {
			t.Fatalf("CreateProfileForAccount() error = %v", err)
		}
		if profile.AccountID != 1001 {
			t.Fatalf("profile account id = %d, want 1001", profile.AccountID)
		}
		if profile.PublicID != "user_pub_1" {
			t.Fatalf("profile public id = %q, want user_pub_1", profile.PublicID)
		}
		if profile.Nickname != "Alice" {
			t.Fatalf("profile nickname = %q, want Alice", profile.Nickname)
		}
		if profile.Status != domain.UserStatusActive {
			t.Fatalf("profile status = %q, want %q", profile.Status, domain.UserStatusActive)
		}
		if profile.ProfileVersion != 0 {
			t.Fatalf("profile version = %d, want 0", profile.ProfileVersion)
		}

		stored, err := service.GetMyProfile(context.Background(), profile.UserID)
		if err != nil {
			t.Fatalf("GetMyProfile() error = %v", err)
		}
		if stored.UserID != profile.UserID {
			t.Fatalf("GetMyProfile() user id = %d, want %d", stored.UserID, profile.UserID)
		}

		publicProfile, err := service.GetUserProfileByPublicID(context.Background(), profile.PublicID)
		if err != nil {
			t.Fatalf("GetUserProfileByPublicID() error = %v", err)
		}
		if publicProfile.PublicID != profile.PublicID {
			t.Fatalf("GetUserProfileByPublicID() public id = %q, want %q", publicProfile.PublicID, profile.PublicID)
		}
		if len(outbox.messages) != 1 {
			t.Fatalf("outbox message count = %d, want 1", len(outbox.messages))
		}
		if outbox.messages[0].EventType != "user.profile.created" {
			t.Fatalf("outbox event type = %q, want user.profile.created", outbox.messages[0].EventType)
		}
		var payload map[string]any
		if err := json.Unmarshal(outbox.messages[0].Payload, &payload); err != nil {
			t.Fatalf("json.Unmarshal(payload) error = %v", err)
		}
		if got := payload["profileVersion"]; got != float64(0) {
			t.Fatalf("payload profileVersion = %v, want 0", got)
		}
	})

	t.Run("returns existing profile for duplicate account without new id or event", func(t *testing.T) {
		store := newFakeProfileStore()
		existing := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 88,
			PublicID:               "user_pub_existing",
			AccountID:              2002,
			Nickname:               "Existing",
			Bio:                    "hello",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusActive,
			ProfileVersion:         3,
			CreatedAt:              now.Add(-time.Hour),
			UpdatedAt:              now.Add(-time.Hour),
		})
		outbox := &fakeOutboxPublisher{}
		txRunner := &fakeTransactionRunner{}
		idGenerator := &fakePublicIDGenerator{ids: []domain.PublicID{"should-not-be-used"}}
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: idGenerator,
			Outbox: outbox, TxRunner: txRunner, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})

		profile, err := service.CreateProfileForAccount(context.Background(), CreateProfileForAccountCommand{
			AccountID: existing.AccountID,
			Username:  "Changed",
		})
		if err != nil {
			t.Fatalf("CreateProfileForAccount() error = %v", err)
		}
		if profile.UserID != existing.UserID || profile.PublicID != existing.PublicID || profile.ProfileVersion != existing.ProfileVersion {
			t.Fatalf("CreateProfileForAccount() = %#v, want existing %#v", profile, existing)
		}
		if txRunner.calledCount != 0 {
			t.Fatalf("transaction count = %d, want 0", txRunner.calledCount)
		}
		if idGenerator.calls != 0 {
			t.Fatalf("public id generator calls = %d, want 0", idGenerator.calls)
		}
		if len(outbox.messages) != 0 {
			t.Fatalf("outbox message count = %d, want 0", len(outbox.messages))
		}
	})

	t.Run("returns existing profile when conflict-safe create sees concurrent account creation", func(t *testing.T) {
		store := newFakeProfileStore()
		existing := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 89,
			PublicID:               "user_pub_race",
			AccountID:              2003,
			Nickname:               "RaceWinner",
			Bio:                    "hello",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusActive,
			ProfileVersion:         6,
			CreatedAt:              now.Add(-time.Hour),
			UpdatedAt:              now.Add(-time.Hour),
		})
		store.hideAccountLookupOnce(existing.AccountID)
		store.abortReloadAfterConflict(existing.AccountID)
		outbox := &fakeOutboxPublisher{}
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{ids: []domain.PublicID{"user_pub_unused"}},
			Outbox: outbox, TxRunner: &fakeTransactionRunner{}, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})

		profile, err := service.CreateProfileForAccount(context.Background(), CreateProfileForAccountCommand{
			AccountID: existing.AccountID,
			Username:  "RaceWinner",
		})
		if err != nil {
			t.Fatalf("CreateProfileForAccount() error = %v", err)
		}
		if profile != existing {
			t.Fatalf("CreateProfileForAccount() = %#v, want existing %#v", profile, existing)
		}
		if store.createOrGetCalls != 1 || len(outbox.messages) != 0 {
			t.Fatalf("createOrGetCalls=%d outbox=%d", store.createOrGetCalls, len(outbox.messages))
		}
	})

	t.Run("returns nickname taken when default nickname already exists", func(t *testing.T) {
		store := newFakeProfileStore()
		store.seedProfile(t, domain.ProfileSeed{
			UserID:                 91,
			PublicID:               "user_pub_taken",
			AccountID:              3001,
			Nickname:               "Taken",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusActive,
			CreatedAt:              now.Add(-time.Hour),
			UpdatedAt:              now.Add(-time.Hour),
		})
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{ids: []domain.PublicID{"user_pub_new"}},
			Outbox: &fakeOutboxPublisher{}, TxRunner: &fakeTransactionRunner{}, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})
		_, err := service.CreateProfileForAccount(context.Background(), CreateProfileForAccountCommand{
			AccountID: 3002,
			Username:  " Taken ",
		})
		if !errors.Is(err, domain.ErrNicknameTaken) {
			t.Fatalf("CreateProfileForAccount() error = %v, want %v", err, domain.ErrNicknameTaken)
		}
	})

	t.Run("validates nickname inputs before persistence", func(t *testing.T) {
		valid15 := strings.Repeat("界", 15)
		for _, tc := range []struct {
			name, username string
			wantErr        error
		}{{"15 runes valid", " " + valid15 + " ", nil}, {"trimmed empty", " \t ", domain.ErrNicknameInvalid}, {"too long", strings.Repeat("界", 16), domain.ErrNicknameInvalid}, {"contains angle bracket", "Ali<ce", domain.ErrNicknameInvalid}, {"contains newline", "Ali\nce", domain.ErrNicknameInvalid}, {"contains control", "Ali\x07ce", domain.ErrNicknameInvalid}} {
			t.Run(tc.name, func(t *testing.T) {
				store, outbox, txRunner := newFakeProfileStore(), &fakeOutboxPublisher{}, &fakeTransactionRunner{}
				service := mustNewService(t, Dependencies{Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{ids: []domain.PublicID{"user_pub_case"}}, Outbox: outbox, TxRunner: txRunner, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{}})
				profile, err := service.CreateProfileForAccount(context.Background(), CreateProfileForAccountCommand{AccountID: 3100, Username: tc.username})
				if tc.wantErr == nil {
					if err != nil || profile.Nickname != valid15 || store.createCalls != 1 || len(outbox.messages) != 1 {
						t.Fatalf("CreateProfileForAccount() profile=%#v err=%v createCalls=%d outbox=%d", profile, err, store.createCalls, len(outbox.messages))
					}
					return
				}
				if !errors.Is(err, tc.wantErr) || store.createCalls != 0 || txRunner.calledCount != 0 || len(outbox.messages) != 0 {
					t.Fatalf("CreateProfileForAccount() err=%v createCalls=%d txCalls=%d outbox=%d", err, store.createCalls, txRunner.calledCount, len(outbox.messages))
				}
			})
		}
	})

	t.Run("hides deleted profile from public query", func(t *testing.T) {
		store := newFakeProfileStore()
		profile := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 77,
			PublicID:               "user_pub_deleted",
			AccountID:              4004,
			Nickname:               "DeletedUser",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusDeleted,
			CreatedAt:              now.Add(-time.Hour),
			UpdatedAt:              now.Add(-time.Hour),
		})
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{},
			Outbox: &fakeOutboxPublisher{}, TxRunner: &fakeTransactionRunner{}, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})
		_, err := service.GetMyProfile(context.Background(), profile.UserID)
		if !errors.Is(err, domain.ErrProfileNotFound) {
			t.Fatalf("GetMyProfile() error = %v, want %v", err, domain.ErrProfileNotFound)
		}
		_, err = service.GetUserProfileByPublicID(context.Background(), profile.PublicID)
		if !errors.Is(err, domain.ErrProfileNotFound) {
			t.Fatalf("GetUserProfileByPublicID() error = %v, want %v", err, domain.ErrProfileNotFound)
		}
	})
}

func TestUpdateProfileValidatesAndPublishesOnlyForPublicChanges(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)

	t.Run("validates avatar reference before transaction", func(t *testing.T) {
		store := newFakeProfileStore()
		profile := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 501,
			PublicID:               "user_pub_501",
			AccountID:              1501,
			Nickname:               "Alice",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusActive,
			CreatedAt:              now.Add(-time.Hour),
			UpdatedAt:              now.Add(-time.Hour),
		})
		trace := &callTrace{}
		files := &fakeFileReferenceClient{trace: trace, err: domain.ErrAvatarInvalid}
		txRunner := &fakeTransactionRunner{trace: trace}
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: files, IDs: &fakePublicIDGenerator{},
			Outbox: &fakeOutboxPublisher{}, TxRunner: txRunner, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})
		_, err := service.UpdateProfile(context.Background(), UpdateProfileCommand{
			UserID:                 profile.UserID,
			Nickname:               optionalString("Alice"),
			AvatarFileID:           optionalString("avatar-1"),
			Bio:                    optionalString(""),
			StrangerMessageAllowed: optionalBool(true),
		})
		if !errors.Is(err, domain.ErrAvatarInvalid) {
			t.Fatalf("UpdateProfile() error = %v, want %v", err, domain.ErrAvatarInvalid)
		}
		if txRunner.calledCount != 0 {
			t.Fatalf("transaction count = %d, want 0", txRunner.calledCount)
		}
		if got := trace.calls; len(got) != 1 || got[0] != "file.validate" {
			t.Fatalf("call order = %#v, want only file.validate", got)
		}
	})

	t.Run("updates public profile fields increments version and emits event", func(t *testing.T) {
		store := newFakeProfileStore()
		profile := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 502,
			PublicID:               "user_pub_502",
			AccountID:              1502,
			Nickname:               "Alice",
			AvatarFileID:           "avatar-1",
			Bio:                    "old",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusActive,
			ProfileVersion:         4,
			CreatedAt:              now.Add(-2 * time.Hour),
			UpdatedAt:              now.Add(-2 * time.Hour),
		})
		store.publicUpdateVersionDelta = 10
		outbox := &fakeOutboxPublisher{}
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{},
			Outbox: outbox, TxRunner: &fakeTransactionRunner{}, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})
		updated, err := service.UpdateProfile(context.Background(), UpdateProfileCommand{
			UserID:                 profile.UserID,
			Nickname:               optionalString("Alice2"),
			AvatarFileID:           optionalString("avatar-2"),
			Bio:                    optionalString("new bio"),
			StrangerMessageAllowed: optionalBool(false),
		})
		if err != nil {
			t.Fatalf("UpdateProfile() error = %v", err)
		}
		if updated.Nickname != "Alice2" || updated.AvatarFileID != "avatar-2" || updated.Bio != "new bio" {
			t.Fatalf("updated profile = %#v", updated)
		}
		if store.lastPublicUpdateInput.ProfileVersion != 4 {
			t.Fatalf("repository input profile version = %d, want 4 before atomic increment", store.lastPublicUpdateInput.ProfileVersion)
		}
		if updated.ProfileVersion != 15 {
			t.Fatalf("updated profile version = %d, want 15", updated.ProfileVersion)
		}
		if len(outbox.messages) != 1 {
			t.Fatalf("outbox message count = %d, want 1", len(outbox.messages))
		}
		if outbox.messages[0].EventType != "user.profile.updated" {
			t.Fatalf("outbox event type = %q, want user.profile.updated", outbox.messages[0].EventType)
		}

		var payload map[string]any
		if err := json.Unmarshal(outbox.messages[0].Payload, &payload); err != nil {
			t.Fatalf("json.Unmarshal(payload) error = %v", err)
		}
		if got := payload["profileVersion"]; got != float64(15) {
			t.Fatalf("payload profileVersion = %v, want 15", got)
		}
		if got := payload["nickname"]; got != "Alice2" {
			t.Fatalf("payload nickname = %v, want Alice2", got)
		}
	})

	t.Run("deleted profile is treated as not found", func(t *testing.T) {
		store := newFakeProfileStore()
		profile := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 509,
			PublicID:               "user_pub_deleted_patch",
			AccountID:              1509,
			Nickname:               "DeletedPatch",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusDeleted,
			CreatedAt:              now.Add(-time.Hour),
			UpdatedAt:              now.Add(-time.Hour),
		})
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{},
			Outbox: &fakeOutboxPublisher{}, TxRunner: &fakeTransactionRunner{}, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})

		_, err := service.UpdateProfile(context.Background(), UpdateProfileCommand{
			UserID:   profile.UserID,
			Nickname: optionalString("NewName"),
		})
		if !errors.Is(err, domain.ErrProfileNotFound) {
			t.Fatalf("UpdateProfile() error = %v, want %v", err, domain.ErrProfileNotFound)
		}
	})

	t.Run("preserves omitted fields from transaction current snapshot", func(t *testing.T) {
		store := newFakeProfileStore()
		profile := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 5021,
			PublicID:               "user_pub_5021",
			AccountID:              25021,
			Nickname:               "Alice",
			AvatarFileID:           "avatar-from-tx",
			Bio:                    "bio from tx current",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusActive,
			ProfileVersion:         9,
			CreatedAt:              now.Add(-2 * time.Hour),
			UpdatedAt:              now.Add(-2 * time.Hour),
		})
		outbox := &fakeOutboxPublisher{}
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{},
			Outbox: outbox, TxRunner: &fakeTransactionRunner{}, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})

		cmd := UpdateProfileCommand{UserID: profile.UserID, Nickname: optionalString("Alice2")}

		updated, err := service.UpdateProfile(context.Background(), cmd)
		if err != nil {
			t.Fatalf("UpdateProfile() error = %v", err)
		}
		if updated.AvatarFileID != "avatar-from-tx" || updated.Bio != "bio from tx current" || !updated.StrangerMessageAllowed {
			t.Fatalf("UpdateProfile() = %#v, want omitted fields preserved from transaction current", updated)
		}
		if store.lastPublicUpdateInput.AvatarFileID != "avatar-from-tx" || store.lastPublicUpdateInput.Bio != "bio from tx current" || !store.lastPublicUpdateInput.StrangerMessageAllowed {
			t.Fatalf("repository input = %#v, want omitted fields preserved", store.lastPublicUpdateInput)
		}
		if len(outbox.messages) != 1 || outbox.messages[0].EventType != "user.profile.updated" {
			t.Fatalf("outbox messages = %#v, want single user.profile.updated", outbox.messages)
		}
	})

	t.Run("only updates stranger message setting without version bump or event", func(t *testing.T) {
		store := newFakeProfileStore()
		profile := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 503,
			PublicID:               "user_pub_503",
			AccountID:              1503,
			Nickname:               "Alice",
			Bio:                    "old",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusActive,
			ProfileVersion:         7,
			CreatedAt:              now.Add(-2 * time.Hour),
			UpdatedAt:              now.Add(-2 * time.Hour),
		})
		outbox := &fakeOutboxPublisher{}
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{},
			Outbox: outbox, TxRunner: &fakeTransactionRunner{}, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})

		updated, err := service.UpdateProfile(context.Background(), UpdateProfileCommand{
			UserID:                 profile.UserID,
			Nickname:               optionalString(profile.Nickname),
			AvatarFileID:           optionalString(profile.AvatarFileID),
			Bio:                    optionalString(profile.Bio),
			StrangerMessageAllowed: optionalBool(false),
		})
		if err != nil {
			t.Fatalf("UpdateProfile() error = %v", err)
		}
		if updated.ProfileVersion != 7 {
			t.Fatalf("updated profile version = %d, want 7", updated.ProfileVersion)
		}
		if updated.StrangerMessageAllowed {
			t.Fatal("stranger message allowed = true, want false")
		}
		if len(outbox.messages) != 0 {
			t.Fatalf("outbox message count = %d, want 0", len(outbox.messages))
		}
	})

	t.Run("rejects update for non active user", func(t *testing.T) {
		store := newFakeProfileStore()
		profile := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 504,
			PublicID:               "user_pub_504",
			AccountID:              1504,
			Nickname:               "Alice",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusDeactivated,
			CreatedAt:              now.Add(-2 * time.Hour),
			UpdatedAt:              now.Add(-2 * time.Hour),
		})
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{},
			Outbox: &fakeOutboxPublisher{}, TxRunner: &fakeTransactionRunner{}, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})
		_, err := service.UpdateProfile(context.Background(), UpdateProfileCommand{
			UserID:                 profile.UserID,
			Nickname:               optionalString("Alice2"),
			AvatarFileID:           optionalString(""),
			Bio:                    optionalString(""),
			StrangerMessageAllowed: optionalBool(true),
		})
		if !errors.Is(err, domain.ErrUserNotActive) {
			t.Fatalf("UpdateProfile() error = %v, want %v", err, domain.ErrUserNotActive)
		}
	})

	t.Run("validates nickname and bio inputs without writes", func(t *testing.T) {
		valid15, validBio100 := strings.Repeat("界", 15), strings.Repeat("a", 100)
		for _, tc := range []struct {
			name, nickname, bio string
			wantErr             error
		}{{"nickname 15 runes valid", valid15, "old", nil}, {"nickname trimmed empty", " \t ", "old", domain.ErrNicknameInvalid}, {"nickname too long", strings.Repeat("界", 16), "old", domain.ErrNicknameInvalid}, {"nickname contains angle bracket", "Ali>ce", "old", domain.ErrNicknameInvalid}, {"nickname contains newline", "Ali\nce", "old", domain.ErrNicknameInvalid}, {"nickname contains control", "Ali\x07ce", "old", domain.ErrNicknameInvalid}, {"bio 100 runes valid", "Alice", validBio100, nil}, {"bio newlines valid", "Alice", "line1\nline2", nil}, {"bio too long", "Alice", strings.Repeat("a", 101), domain.ErrBioInvalid}, {"bio contains angle bracket", "Alice", "bio<script>", domain.ErrBioInvalid}, {"bio contains tab", "Alice", "bio\ttext", domain.ErrBioInvalid}, {"bio contains carriage return", "Alice", "line1\rline2", domain.ErrBioInvalid}, {"bio contains control", "Alice", "bio\x07", domain.ErrBioInvalid}} {
			t.Run(tc.name, func(t *testing.T) {
				store := newFakeProfileStore()
				before := store.seedProfile(t, domain.ProfileSeed{UserID: 550, PublicID: "user_pub_550", AccountID: 1550, Nickname: "Alice", Bio: "old", StrangerMessageAllowed: true, Status: domain.UserStatusActive, ProfileVersion: 2, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)})
				outbox := &fakeOutboxPublisher{}
				service := mustNewService(t, Dependencies{Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{}, Outbox: outbox, TxRunner: &fakeTransactionRunner{}, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{}})
				updated, err := service.UpdateProfile(context.Background(), UpdateProfileCommand{
					UserID:                 before.UserID,
					Nickname:               optionalString(tc.nickname),
					AvatarFileID:           optionalString(before.AvatarFileID),
					Bio:                    optionalString(tc.bio),
					StrangerMessageAllowed: optionalBool(before.StrangerMessageAllowed),
				})
				if tc.wantErr == nil {
					if err != nil || store.updateCalls != 0 || store.updatePublicCalls != 1 || len(outbox.messages) != 1 || updated.Nickname != strings.TrimSpace(tc.nickname) || updated.Bio != tc.bio {
						t.Fatalf("UpdateProfile() updated=%#v err=%v updateCalls=%d publicUpdateCalls=%d outbox=%d", updated, err, store.updateCalls, store.updatePublicCalls, len(outbox.messages))
					}
					return
				}
				after, _ := store.GetByUserID(context.Background(), before.UserID)
				if !errors.Is(err, tc.wantErr) || after != before || store.updateCalls != 0 || store.updatePublicCalls != 0 || len(outbox.messages) != 0 {
					t.Fatalf("UpdateProfile() err=%v after=%#v updateCalls=%d publicUpdateCalls=%d outbox=%d", err, after, store.updateCalls, store.updatePublicCalls, len(outbox.messages))
				}
			})
		}
	})
}

func TestProfileCacheInvalidationRecordsDeleteFailures(t *testing.T) {
	now := time.Date(2026, 7, 4, 13, 0, 0, 0, time.UTC)
	store := newFakeProfileStore()
	profile := store.seedProfile(t, domain.ProfileSeed{
		UserID:                 701,
		PublicID:               "user_pub_701",
		AccountID:              1701,
		Nickname:               "CacheUser",
		StrangerMessageAllowed: true,
		Status:                 domain.UserStatusActive,
		CreatedAt:              now.Add(-time.Hour),
		UpdatedAt:              now.Add(-time.Hour),
	})
	cacheErr := errors.New("redis unavailable")
	recorder := &fakeCacheFailureRecorder{}
	service := mustNewService(t, Dependencies{
		Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{},
		Outbox: &fakeOutboxPublisher{}, TxRunner: &fakeTransactionRunner{}, Clock: fixedClock{now: now},
		Cache: &fakeCacheStore{err: cacheErr}, CacheFailures: recorder,
	})

	_, err := service.UpdateProfile(context.Background(), UpdateProfileCommand{
		UserID:                 profile.UserID,
		StrangerMessageAllowed: optionalBool(false),
	})
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if len(recorder.failures) != 1 {
		t.Fatalf("cache failure count = %d, want 1", len(recorder.failures))
	}
	failure := recorder.failures[0]
	if failure.operation != "user.profile.invalidate" || !errors.Is(failure.err, cacheErr) {
		t.Fatalf("cache failure = %#v, want operation user.profile.invalidate and cache error", failure)
	}
	if len(failure.keys) != 4 {
		t.Fatalf("cache failure keys = %#v, want 4 keys", failure.keys)
	}
}

func TestUserStatusTransitionsAreIdempotentAndPublishEvents(t *testing.T) {
	now := time.Date(2026, 7, 4, 11, 0, 0, 0, time.UTC)

	t.Run("deactivate is idempotent by account id", func(t *testing.T) {
		store := newFakeProfileStore()
		trace := &callTrace{}
		store.trace = trace
		profile := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 601,
			PublicID:               "user_pub_601",
			AccountID:              2601,
			Nickname:               "Alice",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusActive,
			CreatedAt:              now.Add(-time.Hour),
			UpdatedAt:              now.Add(-time.Hour),
		})
		outbox := &fakeOutboxPublisher{}
		txRunner := &fakeTransactionRunner{trace: trace}
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{},
			Outbox: outbox, TxRunner: txRunner, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})
		updated, err := service.DeactivateUserProfile(context.Background(), DeactivateUserProfileCommand{AccountID: profile.AccountID})
		if err != nil {
			t.Fatalf("DeactivateUserProfile() error = %v", err)
		}
		if updated.Status != domain.UserStatusDeactivated {
			t.Fatalf("deactivated status = %q, want %q", updated.Status, domain.UserStatusDeactivated)
		}
		if len(outbox.messages) != 1 || outbox.messages[0].EventType != "user.deactivated" {
			t.Fatalf("outbox messages = %#v, want single user.deactivated", outbox.messages)
		}

		updated, err = service.DeactivateUserProfile(context.Background(), DeactivateUserProfileCommand{AccountID: profile.AccountID})
		if err != nil {
			t.Fatalf("DeactivateUserProfile() second call error = %v", err)
		}
		if updated.Status != domain.UserStatusDeactivated {
			t.Fatalf("deactivated status second call = %q, want %q", updated.Status, domain.UserStatusDeactivated)
		}
		if len(outbox.messages) != 1 {
			t.Fatalf("outbox message count after second call = %d, want 1", len(outbox.messages))
		}
		if txRunner.calledCount != 2 || strings.Join(trace.calls, ",") != "tx.start,repo.deactivate_by_account,tx.start,repo.deactivate_by_account" {
			t.Fatalf("tx=%d trace=%q", txRunner.calledCount, strings.Join(trace.calls, ","))
		}
	})

	t.Run("deactivate rejects deleted user", func(t *testing.T) {
		store := newFakeProfileStore()
		profile := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 603,
			PublicID:               "user_pub_603",
			AccountID:              2603,
			Nickname:               "DeletedAlice",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusDeleted,
			CreatedAt:              now.Add(-time.Hour),
			UpdatedAt:              now.Add(-time.Hour),
		})
		outbox := &fakeOutboxPublisher{}
		trace := &callTrace{}
		store.trace = trace
		txRunner := &fakeTransactionRunner{trace: trace}
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{},
			Outbox: outbox, TxRunner: txRunner, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})

		_, err := service.DeactivateUserProfile(context.Background(), DeactivateUserProfileCommand{AccountID: profile.AccountID})
		if !errors.Is(err, domain.ErrInvalidStatusTransition) {
			t.Fatalf("DeactivateUserProfile() error = %v, want %v", err, domain.ErrInvalidStatusTransition)
		}
		if txRunner.calledCount != 1 || strings.Join(trace.calls, ",") != "tx.start,repo.deactivate_by_account" {
			t.Fatalf("tx=%d trace=%q", txRunner.calledCount, strings.Join(trace.calls, ","))
		}
		if len(outbox.messages) != 0 {
			t.Fatalf("outbox message count = %d, want 0", len(outbox.messages))
		}
	})

	t.Run("mark deleted and restore are idempotent", func(t *testing.T) {
		store := newFakeProfileStore()
		trace := &callTrace{}
		store.trace = trace
		profile := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 602,
			PublicID:               "user_pub_602",
			AccountID:              2602,
			Nickname:               "Bob",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusDeactivated,
			CreatedAt:              now.Add(-time.Hour),
			UpdatedAt:              now.Add(-time.Hour),
		})
		outbox := &fakeOutboxPublisher{}
		txRunner := &fakeTransactionRunner{trace: trace}
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{},
			Outbox: outbox, TxRunner: txRunner, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})

		deleted, err := service.MarkUserDeleted(context.Background(), MarkUserDeletedCommand{
			UserID:     profile.UserID,
			OperatorID: 9001,
			Reason:     "compliance",
		})
		if err != nil {
			t.Fatalf("MarkUserDeleted() error = %v", err)
		}
		if deleted.Status != domain.UserStatusDeleted {
			t.Fatalf("deleted status = %q, want %q", deleted.Status, domain.UserStatusDeleted)
		}
		if deleted.DeletedBy != 9001 || deleted.DeletedReason != "compliance" {
			t.Fatalf("deleted metadata = %#v", deleted)
		}
		if len(outbox.messages) != 1 || outbox.messages[0].EventType != "user.deleted" {
			t.Fatalf("outbox messages after delete = %#v, want single user.deleted", outbox.messages)
		}

		deleted, err = service.MarkUserDeleted(context.Background(), MarkUserDeletedCommand{
			UserID:     profile.UserID,
			OperatorID: 9002,
			Reason:     "ignored",
		})
		if err != nil {
			t.Fatalf("MarkUserDeleted() second call error = %v", err)
		}
		if len(outbox.messages) != 1 {
			t.Fatalf("outbox message count after second delete = %d, want 1", len(outbox.messages))
		}

		restored, err := service.RestoreDeletedUserProfile(context.Background(), RestoreDeletedUserProfileCommand{
			UserID:     profile.UserID,
			OperatorID: 9003,
			Reason:     "appeal",
		})
		if err != nil {
			t.Fatalf("RestoreDeletedUserProfile() error = %v", err)
		}
		if restored.Status != domain.UserStatusActive {
			t.Fatalf("restored status = %q, want %q", restored.Status, domain.UserStatusActive)
		}
		if restored.RestoredBy != 9003 || restored.RestoredReason != "appeal" {
			t.Fatalf("restored metadata = %#v", restored)
		}
		if len(outbox.messages) != 2 || outbox.messages[1].EventType != "user.restored" {
			t.Fatalf("outbox messages after restore = %#v, want user.restored appended", outbox.messages)
		}

		restored, err = service.RestoreDeletedUserProfile(context.Background(), RestoreDeletedUserProfileCommand{
			UserID:     profile.UserID,
			OperatorID: 9004,
			Reason:     "ignored",
		})
		if err != nil {
			t.Fatalf("RestoreDeletedUserProfile() second call error = %v", err)
		}
		if len(outbox.messages) != 2 {
			t.Fatalf("outbox message count after second restore = %d, want 2", len(outbox.messages))
		}
		if txRunner.calledCount != 4 || strings.Join(trace.calls, ",") != "tx.start,repo.mark_deleted,tx.start,repo.mark_deleted,tx.start,repo.restore_deleted,tx.start,repo.restore_deleted" {
			t.Fatalf("tx=%d trace=%q", txRunner.calledCount, strings.Join(trace.calls, ","))
		}
	})

	t.Run("propagates repository transition guard without publishing event", func(t *testing.T) {
		store := newFakeProfileStore()
		trace := &callTrace{}
		store.trace = trace
		profile := store.seedProfile(t, domain.ProfileSeed{
			UserID:                 604,
			PublicID:               "user_pub_604",
			AccountID:              2604,
			Nickname:               "Carol",
			StrangerMessageAllowed: true,
			Status:                 domain.UserStatusActive,
			CreatedAt:              now.Add(-time.Hour),
			UpdatedAt:              now.Add(-time.Hour),
		})
		store.markDeletedErrors[profile.UserID] = domain.ErrInvalidStatusTransition
		outbox := &fakeOutboxPublisher{}
		txRunner := &fakeTransactionRunner{trace: trace}
		service := mustNewService(t, Dependencies{
			Profiles: store, Queries: store, Files: &fakeFileReferenceClient{}, IDs: &fakePublicIDGenerator{},
			Outbox: outbox, TxRunner: txRunner, Clock: fixedClock{now: now}, Cache: &fakeCacheStore{},
		})

		_, err := service.MarkUserDeleted(context.Background(), MarkUserDeletedCommand{UserID: profile.UserID, OperatorID: 9005, Reason: "late compliance"})
		if !errors.Is(err, domain.ErrInvalidStatusTransition) || len(outbox.messages) != 0 || strings.Join(trace.calls, ",") != "tx.start,repo.mark_deleted" {
			t.Fatalf("err=%v outbox=%d trace=%q", err, len(outbox.messages), strings.Join(trace.calls, ","))
		}
	})
}
