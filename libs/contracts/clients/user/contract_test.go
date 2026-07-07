package user

import (
	"encoding/json"
	"testing"
)

func TestInternalHTTPContractPaths(t *testing.T) {
	tests := map[string]string{
		"availability": BatchAvailabilityPath,
		"simple":       BatchSimplePath,
		"blocked":      BatchCheckBlockedPath,
		"followers":    ListFollowerShardPath,
	}
	for name, path := range tests {
		if path == "" || path[0] != '/' {
			t.Fatalf("%s path = %q, want absolute path", name, path)
		}
	}
}

func TestListFollowerShardContractJSONShape(t *testing.T) {
	payload, err := json.Marshal(ListFollowerShardRequest{
		FollowingID:   77,
		AudienceClass: "HOT",
		ActiveSince:   "2026-07-01T00:00:00Z",
		Cursor:        "42",
		Limit:         200,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(payload) != `{"followingId":77,"audienceClass":"HOT","activeSince":"2026-07-01T00:00:00Z","cursor":"42","limit":200}` {
		t.Fatalf("payload = %s", payload)
	}

	response, err := json.Marshal(ListFollowerShardResponse{
		FollowerIDs: []int64{101, 99},
		NextCursor:  "99",
		HasMore:     true,
	})
	if err != nil {
		t.Fatalf("Marshal(response) error = %v", err)
	}
	want := `{"followerIds":[101,99],"nextCursor":"99","hasMore":true}`
	if string(response) != want {
		t.Fatalf("response = %s, want %s", response, want)
	}
}

func TestIDsRequestJSONShape(t *testing.T) {
	payload, err := json.Marshal(IDsRequest{UserIDs: []int64{42, 77}})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(payload) != `{"userIds":[42,77]}` {
		t.Fatalf("payload = %s", payload)
	}
}

func TestBlockPairsRequestJSONShape(t *testing.T) {
	payload, err := json.Marshal(BlockPairsRequest{
		Pairs: []BlockPair{{BlockerID: 42, BlockedID: 77}},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(payload) != `{"pairs":[{"blockerId":42,"blockedId":77}]}` {
		t.Fatalf("payload = %s", payload)
	}
}

func TestSimpleBatchResponseJSONShape(t *testing.T) {
	payload, err := json.Marshal(SimpleBatchResponse{
		Items: []SimpleUser{{
			UserID:         77,
			PublicID:       "user_pub_77",
			Nickname:       "Alice",
			AvatarFileID:   "avatar_1",
			AvatarURL:      "https://cdn.example/avatar_1",
			ProfileVersion: 3,
			Status:         "ACTIVE",
		}},
		MissingUserIDs: []int64{88},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"items":[{"userId":77,"publicId":"user_pub_77","nickname":"Alice","avatarFileId":"avatar_1","avatarUrl":"https://cdn.example/avatar_1","profileVersion":3,"status":"ACTIVE"}],"missingUserIds":[88]}`
	if string(payload) != want {
		t.Fatalf("payload = %s", payload)
	}
}
