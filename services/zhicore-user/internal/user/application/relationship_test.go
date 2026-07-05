package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/domain"
)

func TestBlockUserRemovesMutualFollowsAndPublishesEvents(t *testing.T) {
	now := time.Date(2026, 7, 4, 14, 0, 0, 0, time.UTC)
	store, relationships, outbox := newFakeProfileStore(), newFakeRelationshipStore(), &fakeOutboxPublisher{}
	actor := seedRelationshipProfile(t, store, 101, "user_pub_actor", domain.UserStatusActive)
	target := seedRelationshipProfile(t, store, 202, "user_pub_target", domain.UserStatusActive)
	relationships.seedFollow(actor.UserID, target.UserID, now.Add(-time.Minute))
	relationships.seedFollow(target.UserID, actor.UserID, now.Add(-time.Minute))
	service := mustNewRelationshipService(t, store, relationships, outbox, now)

	if err := service.BlockUser(context.Background(), BlockUserCommand{ActorUserID: UserID(actor.UserID), TargetPublicID: PublicID(target.PublicID), Reason: "spam"}); err != nil {
		t.Fatalf("BlockUser() error = %v", err)
	}

	if !relationships.hasBlock(actor.UserID, target.UserID) {
		t.Fatal("expected block relation to be stored")
	}
	if relationships.hasFollow(actor.UserID, target.UserID) || relationships.hasFollow(target.UserID, actor.UserID) {
		t.Fatalf("block should remove both follow directions, follows=%v", relationships.follows)
	}
	assertFollowStats(t, relationships, actor.UserID, 0, 0)
	assertFollowStats(t, relationships, target.UserID, 0, 0)
	assertEventTypes(t, outbox.messages, []string{"user.blocked", "user.unfollowed", "user.unfollowed"})
	assertEventPayloadField(t, outbox.messages[1], "reason", string(domain.UnfollowReasonBlocked))

	if err := service.BlockUser(context.Background(), BlockUserCommand{ActorUserID: UserID(actor.UserID), TargetPublicID: PublicID(target.PublicID)}); err != nil {
		t.Fatalf("duplicate BlockUser() error = %v", err)
	}
	if len(outbox.messages) != 3 {
		t.Fatalf("duplicate block published %d events, want unchanged 3", len(outbox.messages))
	}
}

func TestBlockUserValidatesSelfAndActiveProfiles(t *testing.T) {
	now := time.Date(2026, 7, 4, 14, 30, 0, 0, time.UTC)

	t.Run("rejects self block", func(t *testing.T) {
		store, relationships, outbox := newFakeProfileStore(), newFakeRelationshipStore(), &fakeOutboxPublisher{}
		actor := seedRelationshipProfile(t, store, 101, "user_pub_actor", domain.UserStatusActive)
		service := mustNewRelationshipService(t, store, relationships, outbox, now)
		err := service.BlockUser(context.Background(), BlockUserCommand{ActorUserID: UserID(actor.UserID), TargetPublicID: PublicID(actor.PublicID)})
		if !errors.Is(err, domain.ErrCannotBlockSelf) {
			t.Fatalf("BlockUser() error = %v, want %v", err, domain.ErrCannotBlockSelf)
		}
		if len(outbox.messages) != 0 || len(relationships.blocks) != 0 {
			t.Fatalf("self block should not mutate relationships or outbox")
		}
	})

	t.Run("requires active actor and target for new block", func(t *testing.T) {
		for _, tc := range []struct {
			name         string
			actorStatus  domain.UserStatus
			targetStatus domain.UserStatus
		}{
			{name: "actor deactivated", actorStatus: domain.UserStatusDeactivated, targetStatus: domain.UserStatusActive},
			{name: "target deleted", actorStatus: domain.UserStatusActive, targetStatus: domain.UserStatusDeleted},
		} {
			t.Run(tc.name, func(t *testing.T) {
				store, relationships, outbox := newFakeProfileStore(), newFakeRelationshipStore(), &fakeOutboxPublisher{}
				actor := seedRelationshipProfile(t, store, 101, "user_pub_actor", tc.actorStatus)
				target := seedRelationshipProfile(t, store, 202, "user_pub_target", tc.targetStatus)
				service := mustNewRelationshipService(t, store, relationships, outbox, now)
				err := service.BlockUser(context.Background(), BlockUserCommand{ActorUserID: UserID(actor.UserID), TargetPublicID: PublicID(target.PublicID)})
				if !errors.Is(err, domain.ErrUserNotActive) {
					t.Fatalf("BlockUser() error = %v, want %v", err, domain.ErrUserNotActive)
				}
				if len(outbox.messages) != 0 || len(relationships.blocks) != 0 {
					t.Fatalf("inactive block should not mutate relationships or outbox")
				}
			})
		}
	})
}

func TestUnblockUserIsIdempotentAndDoesNotRestoreFollows(t *testing.T) {
	now := time.Date(2026, 7, 4, 15, 0, 0, 0, time.UTC)
	store, relationships, outbox := newFakeProfileStore(), newFakeRelationshipStore(), &fakeOutboxPublisher{}
	actor := seedRelationshipProfile(t, store, 101, "user_pub_actor", domain.UserStatusActive)
	target := seedRelationshipProfile(t, store, 202, "user_pub_target", domain.UserStatusDeleted)
	relationships.seedBlock(actor.UserID, target.UserID, now.Add(-time.Hour))
	service := mustNewRelationshipService(t, store, relationships, outbox, now)

	if err := service.UnblockUser(context.Background(), UnblockUserCommand{ActorUserID: UserID(actor.UserID), TargetPublicID: PublicID(target.PublicID)}); err != nil {
		t.Fatalf("UnblockUser() error = %v", err)
	}
	if relationships.hasBlock(actor.UserID, target.UserID) {
		t.Fatal("expected block relation to be removed")
	}
	assertEventTypes(t, outbox.messages, []string{"user.unblocked"})

	if err := service.UnblockUser(context.Background(), UnblockUserCommand{ActorUserID: UserID(actor.UserID), TargetPublicID: PublicID(target.PublicID)}); err != nil {
		t.Fatalf("duplicate UnblockUser() error = %v", err)
	}
	if len(outbox.messages) != 1 || relationships.hasFollow(actor.UserID, target.UserID) {
		t.Fatalf("duplicate unblock should be idempotent and not restore follow")
	}
}

func TestFollowUserHonorsIdempotencyStatsAndBlockGuards(t *testing.T) {
	now := time.Date(2026, 7, 4, 16, 0, 0, 0, time.UTC)
	store, relationships, outbox := newFakeProfileStore(), newFakeRelationshipStore(), &fakeOutboxPublisher{}
	actor := seedRelationshipProfile(t, store, 101, "user_pub_actor", domain.UserStatusActive)
	target := seedRelationshipProfile(t, store, 202, "user_pub_target", domain.UserStatusActive)
	service := mustNewRelationshipService(t, store, relationships, outbox, now)

	if err := service.FollowUser(context.Background(), FollowUserCommand{ActorUserID: UserID(actor.UserID), TargetPublicID: PublicID(target.PublicID)}); err != nil {
		t.Fatalf("FollowUser() error = %v", err)
	}
	if !relationships.hasFollow(actor.UserID, target.UserID) {
		t.Fatal("expected follow relation to be stored")
	}
	assertFollowStats(t, relationships, actor.UserID, 0, 1)
	assertFollowStats(t, relationships, target.UserID, 1, 0)
	assertEventTypes(t, outbox.messages, []string{"user.followed"})

	if err := service.FollowUser(context.Background(), FollowUserCommand{ActorUserID: UserID(actor.UserID), TargetPublicID: PublicID(target.PublicID)}); err != nil {
		t.Fatalf("duplicate FollowUser() error = %v", err)
	}
	if len(outbox.messages) != 1 {
		t.Fatalf("duplicate follow published %d events, want unchanged 1", len(outbox.messages))
	}

	relationships.seedBlock(target.UserID, actor.UserID, now)
	err := service.FollowUser(context.Background(), FollowUserCommand{ActorUserID: UserID(target.UserID), TargetPublicID: PublicID(actor.PublicID)})
	if !errors.Is(err, domain.ErrInteractionBlocked) {
		t.Fatalf("FollowUser() blocked error = %v, want %v", err, domain.ErrInteractionBlocked)
	}

	err = service.FollowUser(context.Background(), FollowUserCommand{ActorUserID: UserID(actor.UserID), TargetPublicID: PublicID(actor.PublicID)})
	if !errors.Is(err, domain.ErrCannotFollowSelf) {
		t.Fatalf("FollowUser() self error = %v, want %v", err, domain.ErrCannotFollowSelf)
	}
}

func TestUnfollowUserIsIdempotentAndPublishesUserRequestReason(t *testing.T) {
	now := time.Date(2026, 7, 4, 17, 0, 0, 0, time.UTC)
	store, relationships, outbox := newFakeProfileStore(), newFakeRelationshipStore(), &fakeOutboxPublisher{}
	actor := seedRelationshipProfile(t, store, 101, "user_pub_actor", domain.UserStatusActive)
	target := seedRelationshipProfile(t, store, 202, "user_pub_target", domain.UserStatusDeleted)
	relationships.seedFollow(actor.UserID, target.UserID, now.Add(-time.Hour))
	service := mustNewRelationshipService(t, store, relationships, outbox, now)

	if err := service.UnfollowUser(context.Background(), UnfollowUserCommand{ActorUserID: UserID(actor.UserID), TargetPublicID: PublicID(target.PublicID)}); err != nil {
		t.Fatalf("UnfollowUser() error = %v", err)
	}
	if relationships.hasFollow(actor.UserID, target.UserID) {
		t.Fatal("expected follow relation to be removed")
	}
	assertFollowStats(t, relationships, actor.UserID, 0, 0)
	assertFollowStats(t, relationships, target.UserID, 0, 0)
	assertEventTypes(t, outbox.messages, []string{"user.unfollowed"})
	assertEventPayloadField(t, outbox.messages[0], "reason", string(domain.UnfollowReasonUserRequest))

	if err := service.UnfollowUser(context.Background(), UnfollowUserCommand{ActorUserID: UserID(actor.UserID), TargetPublicID: PublicID(target.PublicID)}); err != nil {
		t.Fatalf("duplicate UnfollowUser() error = %v", err)
	}
	if len(outbox.messages) != 1 {
		t.Fatalf("duplicate unfollow published %d events, want unchanged 1", len(outbox.messages))
	}
}

func TestRelationshipQueriesUseInternalIDsAndCursorPages(t *testing.T) {
	now := time.Date(2026, 7, 4, 18, 0, 0, 0, time.UTC)
	store, relationships, outbox := newFakeProfileStore(), newFakeRelationshipStore(), &fakeOutboxPublisher{}
	actor := seedRelationshipProfile(t, store, 101, "user_pub_actor", domain.UserStatusActive)
	targetA := seedRelationshipProfile(t, store, 202, "user_pub_target_a", domain.UserStatusActive)
	targetB := seedRelationshipProfile(t, store, 303, "user_pub_target_b", domain.UserStatusActive)
	relationships.seedBlock(actor.UserID, targetA.UserID, now.Add(-2*time.Minute))
	relationships.seedBlock(actor.UserID, targetB.UserID, now.Add(-time.Minute))
	relationships.seedFollow(targetA.UserID, actor.UserID, now.Add(-2*time.Minute))
	relationships.seedFollow(actor.UserID, targetB.UserID, now.Add(-time.Minute))
	service := mustNewRelationshipService(t, store, relationships, outbox, now)

	blocked, err := service.ListBlockedUsers(context.Background(), ListBlockedUsersQuery{ActorUserID: UserID(actor.UserID), Limit: 1})
	if err != nil {
		t.Fatalf("ListBlockedUsers() error = %v", err)
	}
	if len(blocked.Items) != 1 || blocked.Items[0].UserID != UserID(targetB.UserID) || !blocked.HasMore || blocked.NextCursor == "" {
		t.Fatalf("blocked page = %#v, want targetB with next cursor", blocked)
	}

	followers, err := service.ListFollowers(context.Background(), ListFollowersQuery{TargetPublicID: PublicID(actor.PublicID), Limit: 10})
	if err != nil {
		t.Fatalf("ListFollowers() error = %v", err)
	}
	if len(followers.Items) != 1 || followers.Items[0].UserID != UserID(targetA.UserID) {
		t.Fatalf("followers page = %#v, want targetA", followers)
	}

	following, err := service.ListFollowing(context.Background(), ListFollowingQuery{TargetPublicID: PublicID(actor.PublicID), Limit: 10})
	if err != nil {
		t.Fatalf("ListFollowing() error = %v", err)
	}
	if len(following.Items) != 1 || following.Items[0].UserID != UserID(targetB.UserID) {
		t.Fatalf("following page = %#v, want targetB", following)
	}

	checked, err := service.BatchCheckBlocked(context.Background(), []UserPair{{ActorID: UserID(actor.UserID), TargetID: UserID(targetA.UserID)}, {ActorID: UserID(targetA.UserID), TargetID: UserID(targetB.UserID)}})
	if err != nil {
		t.Fatalf("BatchCheckBlocked() error = %v", err)
	}
	if !checked[UserPair{ActorID: UserID(actor.UserID), TargetID: UserID(targetA.UserID)}] || checked[UserPair{ActorID: UserID(targetA.UserID), TargetID: UserID(targetB.UserID)}] {
		t.Fatalf("BatchCheckBlocked() = %#v", checked)
	}

	isFollowing, err := service.CheckFollowing(context.Background(), actor.UserID, targetB.UserID)
	if err != nil {
		t.Fatalf("CheckFollowing() error = %v", err)
	}
	if !isFollowing {
		t.Fatal("CheckFollowing() = false, want true")
	}

	_, err = service.ListBlockedUsers(context.Background(), ListBlockedUsersQuery{ActorUserID: UserID(actor.UserID), Cursor: "not-a-cursor", Limit: 10})
	if !errors.Is(err, domain.ErrCursorInvalid) {
		t.Fatalf("ListBlockedUsers() cursor error = %v, want %v", err, domain.ErrCursorInvalid)
	}
}
