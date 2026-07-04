package domain

import (
	"strconv"
	"strings"
)

const (
	DefaultRelationshipPageLimit = 20
	MaxRelationshipPageLimit     = 100
)

type UnfollowReason string

const (
	UnfollowReasonUserRequest UnfollowReason = "USER_REQUEST"
	UnfollowReasonBlocked     UnfollowReason = "BLOCKED"
)

type UserPair struct {
	ActorID  UserID
	TargetID UserID
}

// RelationshipEvent records relationship facts only; integration names and
// outbox payloads are application-layer contracts.
type RelationshipEvent interface {
	relationshipEvent()
}

type UserFollowed struct {
	FollowerID  UserID
	FollowingID UserID
}

type UserUnfollowed struct {
	FollowerID  UserID
	FollowingID UserID
	Reason      UnfollowReason
}

type UserBlocked struct {
	BlockerID UserID
	BlockedID UserID
	Reason    string
}

type UserUnblocked struct {
	BlockerID UserID
	BlockedID UserID
}

type FollowPlan struct {
	FollowerID  UserID
	FollowingID UserID
}

type UnfollowPlan struct {
	FollowerID  UserID
	FollowingID UserID
}

type BlockPlan struct {
	BlockerID      UserID
	BlockedID      UserID
	Reason         string
	RemovedFollows []UserPair
}

type UnblockPlan struct {
	BlockerID UserID
	BlockedID UserID
}

func PlanFollow(actor, target Profile, blocked bool) (FollowPlan, error) {
	if actor.UserID == target.UserID {
		return FollowPlan{}, ErrCannotFollowSelf
	}
	if actor.Status != UserStatusActive || target.Status != UserStatusActive {
		return FollowPlan{}, ErrUserNotActive
	}
	if blocked {
		return FollowPlan{}, ErrInteractionBlocked
	}
	return FollowPlan{FollowerID: actor.UserID, FollowingID: target.UserID}, nil
}

func PlanUnfollow(actor, target Profile) (UnfollowPlan, error) {
	if actor.UserID == target.UserID {
		return UnfollowPlan{}, ErrCannotFollowSelf
	}
	if err := ensureActiveActor(actor); err != nil {
		return UnfollowPlan{}, err
	}
	return UnfollowPlan{FollowerID: actor.UserID, FollowingID: target.UserID}, nil
}

func PlanBlock(actor, target Profile, reason string) (BlockPlan, error) {
	if actor.UserID == target.UserID {
		return BlockPlan{}, ErrCannotBlockSelf
	}
	if actor.Status != UserStatusActive || target.Status != UserStatusActive {
		return BlockPlan{}, ErrUserNotActive
	}
	return BlockPlan{
		BlockerID: actor.UserID,
		BlockedID: target.UserID,
		Reason:    strings.TrimSpace(reason),
		RemovedFollows: []UserPair{
			{ActorID: actor.UserID, TargetID: target.UserID},
			{ActorID: target.UserID, TargetID: actor.UserID},
		},
	}, nil
}

func PlanUnblock(actor, target Profile) (UnblockPlan, error) {
	if actor.UserID == target.UserID {
		return UnblockPlan{}, ErrCannotBlockSelf
	}
	if err := ensureActiveActor(actor); err != nil {
		return UnblockPlan{}, err
	}
	return UnblockPlan{BlockerID: actor.UserID, BlockedID: target.UserID}, nil
}

func ensureActiveActor(actor Profile) error {
	if actor.Status != UserStatusActive {
		return ErrUserNotActive
	}
	return nil
}

func (p FollowPlan) Event() UserFollowed {
	return UserFollowed{FollowerID: p.FollowerID, FollowingID: p.FollowingID}
}

func (p UnfollowPlan) Event() UserUnfollowed {
	return UserPair{ActorID: p.FollowerID, TargetID: p.FollowingID}.UnfollowedEvent(UnfollowReasonUserRequest)
}

func (p BlockPlan) Event() UserBlocked {
	return UserBlocked{BlockerID: p.BlockerID, BlockedID: p.BlockedID, Reason: p.Reason}
}

func (p UnblockPlan) Event() UserUnblocked {
	return UserUnblocked{BlockerID: p.BlockerID, BlockedID: p.BlockedID}
}

func (p UserPair) UnfollowedEvent(reason UnfollowReason) UserUnfollowed {
	return UserUnfollowed{
		FollowerID:  p.ActorID,
		FollowingID: p.TargetID,
		Reason:      reason,
	}
}

func (UserFollowed) relationshipEvent()   {}
func (UserUnfollowed) relationshipEvent() {}
func (UserBlocked) relationshipEvent()    {}
func (UserUnblocked) relationshipEvent()  {}

func NormalizeRelationshipLimit(limit int) int {
	if limit <= 0 {
		return DefaultRelationshipPageLimit
	}
	if limit > MaxRelationshipPageLimit {
		return MaxRelationshipPageLimit
	}
	return limit
}

func DecodeRelationshipCursor(cursor string) (int64, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || value <= 0 {
		return 0, ErrCursorInvalid
	}
	return value, nil
}

func EncodeRelationshipCursor(id int64) string {
	if id <= 0 {
		return ""
	}
	return strconv.FormatInt(id, 10)
}
