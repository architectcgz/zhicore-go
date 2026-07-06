// Package user contains zhicore-user synchronous client contracts.
package user

const (
	BatchAvailabilityPath = "/api/v1/internal/users/batch-availability"
	BatchSimplePath       = "/api/v1/internal/users/batch-simple"
	BatchCheckBlockedPath = "/api/v1/internal/users/blocks/batch-check"
	ListFollowerShardPath = "/api/v1/internal/users/follower-shard"

	OperationCommentCheckUserAvailability   = "comment.check_user_availability"
	OperationCommentBatchGetAuthorSummaries = "comment.batch_get_author_summaries"
	OperationCommentBatchCheckBlocked       = "comment.batch_check_blocked"
	OperationContentGetOwnerSnapshot        = "content.get_owner_snapshot"
	OperationNotificationListFollowerShard  = "notification.list_follower_shard"
)

type IDsRequest struct {
	UserIDs []int64 `json:"userIds"`
}

type AvailabilityBatchResponse struct {
	Items []AvailabilityItem `json:"items"`
}

type AvailabilityItem struct {
	UserID    int64  `json:"userId"`
	Available bool   `json:"available"`
	Status    string `json:"status"`
}

type SimpleBatchResponse struct {
	Items          []SimpleUser `json:"items"`
	MissingUserIDs []int64      `json:"missingUserIds"`
}

type SimpleUser struct {
	UserID         int64  `json:"userId"`
	PublicID       string `json:"publicId"`
	Nickname       string `json:"nickname"`
	AvatarFileID   string `json:"avatarFileId"`
	AvatarURL      string `json:"avatarUrl,omitempty"`
	ProfileVersion int64  `json:"profileVersion"`
	Status         string `json:"status"`
}

type BlockPairsRequest struct {
	Pairs []BlockPair `json:"pairs"`
}

type BlockPair struct {
	BlockerID int64 `json:"blockerId"`
	BlockedID int64 `json:"blockedId"`
}

type BlockPairsResponse struct {
	Items []BlockPairResult `json:"items"`
}

type BlockPairResult struct {
	BlockerID int64 `json:"blockerId"`
	BlockedID int64 `json:"blockedId"`
	Blocked   bool  `json:"blocked"`
}

type ListFollowerShardRequest struct {
	FollowingID int64  `json:"followingId"`
	Cursor      string `json:"cursor,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type ListFollowerShardResponse struct {
	FollowerIDs []int64 `json:"followerIds"`
	NextCursor  string  `json:"nextCursor,omitempty"`
	HasMore     bool    `json:"hasMore"`
}
