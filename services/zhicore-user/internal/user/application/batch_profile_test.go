package application

import (
	"context"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
)

func TestBatchGetUserSimplePreservesRequestedOrderAndReportsMissingOnce(t *testing.T) {
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	store := newFakeProfileStore()
	store.seedProfile(t, domain.ProfileSeed{
		UserID:                 42,
		PublicID:               "user_pub_42",
		AccountID:              1042,
		Nickname:               "Alice",
		AvatarFileID:           "avatar-42",
		StrangerMessageAllowed: true,
		Status:                 domain.UserStatusActive,
		ProfileVersion:         7,
		CreatedAt:              now,
		UpdatedAt:              now,
	})
	store.seedProfile(t, domain.ProfileSeed{
		UserID:                 77,
		PublicID:               "user_pub_77",
		AccountID:              1077,
		Nickname:               "Bob",
		StrangerMessageAllowed: true,
		Status:                 domain.UserStatusDeactivated,
		ProfileVersion:         3,
		CreatedAt:              now,
		UpdatedAt:              now,
	})
	service := mustNewService(t, Dependencies{
		Profiles: store,
		Queries:  store,
		Files:    &fakeFileReferenceClient{},
		IDs:      &fakePublicIDGenerator{},
		Outbox:   &fakeOutboxPublisher{},
		TxRunner: &fakeTransactionRunner{},
		Clock:    fixedClock{now: now},
		Cache:    &fakeCacheStore{},
	})

	result, err := service.BatchGetUserSimple(context.Background(), []UserID{77, 404, 42, 404})
	if err != nil {
		t.Fatalf("BatchGetUserSimple() error = %v", err)
	}
	if len(result.Items) != 2 || result.Items[0].UserID != 77 || result.Items[1].UserID != 42 {
		t.Fatalf("items = %#v", result.Items)
	}
	if len(result.MissingUserIDs) != 1 || result.MissingUserIDs[0] != 404 {
		t.Fatalf("missing = %#v", result.MissingUserIDs)
	}
}

func TestBatchGetUserAvailabilityMarksMissingAndInactiveUnavailable(t *testing.T) {
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	store := newFakeProfileStore()
	store.seedProfile(t, domain.ProfileSeed{
		UserID:                 42,
		PublicID:               "user_pub_42",
		AccountID:              1042,
		Nickname:               "Alice",
		StrangerMessageAllowed: true,
		Status:                 domain.UserStatusActive,
		ProfileVersion:         7,
		CreatedAt:              now,
		UpdatedAt:              now,
	})
	store.seedProfile(t, domain.ProfileSeed{
		UserID:                 77,
		PublicID:               "user_pub_77",
		AccountID:              1077,
		Nickname:               "Bob",
		StrangerMessageAllowed: true,
		Status:                 domain.UserStatusDeactivated,
		ProfileVersion:         3,
		CreatedAt:              now,
		UpdatedAt:              now,
	})
	service := mustNewService(t, Dependencies{
		Profiles: store,
		Queries:  store,
		Files:    &fakeFileReferenceClient{},
		IDs:      &fakePublicIDGenerator{},
		Outbox:   &fakeOutboxPublisher{},
		TxRunner: &fakeTransactionRunner{},
		Clock:    fixedClock{now: now},
		Cache:    &fakeCacheStore{},
	})

	items, err := service.BatchGetUserAvailability(context.Background(), []UserID{42, 77, 404})
	if err != nil {
		t.Fatalf("BatchGetUserAvailability() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("items = %#v", items)
	}
	if !items[0].Available || items[0].Status != UserStatusActive {
		t.Fatalf("active item = %#v", items[0])
	}
	if items[1].Available || items[1].Status != UserStatusDeactivated {
		t.Fatalf("inactive item = %#v", items[1])
	}
	if items[2].Available || items[2].UserID != 404 || items[2].Status != "" {
		t.Fatalf("missing item = %#v", items[2])
	}
}
