package domain

import (
	"errors"
	"testing"
)

func TestPlanFollowOwnsFollowRulesAndEventIntent(t *testing.T) {
	actor := relationshipProfile(101, UserStatusActive)
	target := relationshipProfile(202, UserStatusActive)

	plan, err := PlanFollow(actor, target, false)
	if err != nil {
		t.Fatalf("PlanFollow() error = %v", err)
	}
	if plan.FollowerID != actor.UserID || plan.FollowingID != target.UserID {
		t.Fatalf("PlanFollow() = %#v", plan)
	}
	event := plan.Event()
	if event != (UserFollowed{FollowerID: actor.UserID, FollowingID: target.UserID}) {
		t.Fatalf("follow event = %#v", event)
	}

	for _, tc := range []struct {
		name    string
		actor   Profile
		target  Profile
		blocked bool
		wantErr error
	}{
		{name: "self", actor: actor, target: actor, wantErr: ErrCannotFollowSelf},
		{name: "inactive target", actor: actor, target: relationshipProfile(202, UserStatusDeleted), wantErr: ErrUserNotActive},
		{name: "blocked", actor: actor, target: target, blocked: true, wantErr: ErrInteractionBlocked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PlanFollow(tc.actor, tc.target, tc.blocked)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("PlanFollow() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestPlanBlockOwnsCleanupAndEventIntent(t *testing.T) {
	actor := relationshipProfile(101, UserStatusActive)
	target := relationshipProfile(202, UserStatusActive)

	plan, err := PlanBlock(actor, target, " spam ")
	if err != nil {
		t.Fatalf("PlanBlock() error = %v", err)
	}
	if plan.BlockerID != actor.UserID || plan.BlockedID != target.UserID || plan.Reason != "spam" {
		t.Fatalf("PlanBlock() = %#v", plan)
	}
	if len(plan.RemovedFollows) != 2 ||
		plan.RemovedFollows[0] != (UserPair{ActorID: actor.UserID, TargetID: target.UserID}) ||
		plan.RemovedFollows[1] != (UserPair{ActorID: target.UserID, TargetID: actor.UserID}) {
		t.Fatalf("removed follows = %#v", plan.RemovedFollows)
	}
	event := plan.Event()
	if event != (UserBlocked{BlockerID: actor.UserID, BlockedID: target.UserID, Reason: "spam"}) {
		t.Fatalf("block event = %#v", event)
	}

	_, err = PlanBlock(actor, actor, "")
	if !errors.Is(err, ErrCannotBlockSelf) {
		t.Fatalf("PlanBlock(self) error = %v, want %v", err, ErrCannotBlockSelf)
	}
}

func TestRelationshipEventsStayAsDomainFacts(t *testing.T) {
	event := (UserPair{ActorID: 101, TargetID: 202}).UnfollowedEvent(UnfollowReasonBlocked)
	if event != (UserUnfollowed{FollowerID: 101, FollowingID: 202, Reason: UnfollowReasonBlocked}) {
		t.Fatalf("unfollow event = %#v", event)
	}
}

func relationshipProfile(userID UserID, status UserStatus) Profile {
	return Profile{
		UserID: userID,
		Status: status,
	}
}
