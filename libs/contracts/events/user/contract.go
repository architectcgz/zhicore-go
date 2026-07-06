package user

import "time"

const (
	EventProfileCreated = "user.profile.created"
	EventProfileUpdated = "user.profile.updated"
	EventDeactivated    = "user.deactivated"
	EventDeleted        = "user.deleted"
	EventRestored       = "user.restored"
	EventFollowed       = "user.followed"
	EventUnfollowed     = "user.unfollowed"
	EventBlocked        = "user.blocked"
	EventUnblocked      = "user.unblocked"
)

type ProfileCreatedPayload struct {
	UserID         int64     `json:"userId"`
	AccountID      int64     `json:"accountId"`
	Nickname       string    `json:"nickname"`
	AvatarFileID   string    `json:"avatarFileId"`
	ProfileVersion int64     `json:"profileVersion"`
	OccurredAt     time.Time `json:"occurredAt"`
}

type ProfileUpdatedPayload struct {
	UserID         int64     `json:"userId"`
	AccountID      int64     `json:"accountId"`
	Nickname       string    `json:"nickname"`
	AvatarFileID   string    `json:"avatarFileId"`
	Bio            string    `json:"bio"`
	ProfileVersion int64     `json:"profileVersion"`
	OccurredAt     time.Time `json:"occurredAt"`
}

type DeactivatedPayload struct {
	UserID     int64     `json:"userId"`
	AccountID  int64     `json:"accountId"`
	OccurredAt time.Time `json:"occurredAt"`
}

type DeletedPayload struct {
	UserID     int64     `json:"userId"`
	OperatorID int64     `json:"operatorId"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurredAt"`
}

type RestoredPayload struct {
	UserID     int64     `json:"userId"`
	OperatorID int64     `json:"operatorId"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurredAt"`
}

type FollowedPayload struct {
	FollowerID  int64     `json:"followerId"`
	FollowingID int64     `json:"followingId"`
	OccurredAt  time.Time `json:"occurredAt"`
}

type UnfollowedPayload struct {
	FollowerID  int64     `json:"followerId"`
	FollowingID int64     `json:"followingId"`
	Reason      string    `json:"reason"`
	OccurredAt  time.Time `json:"occurredAt"`
}

type BlockedPayload struct {
	BlockerID  int64     `json:"blockerId"`
	BlockedID  int64     `json:"blockedId"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurredAt"`
}

type UnblockedPayload struct {
	BlockerID  int64     `json:"blockerId"`
	BlockedID  int64     `json:"blockedId"`
	OccurredAt time.Time `json:"occurredAt"`
}
