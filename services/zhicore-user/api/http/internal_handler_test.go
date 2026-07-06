package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
)

func TestInternalBatchSimpleUsesContractPathAndCallerHeaders(t *testing.T) {
	service := &fakeProfileService{
		batchSimpleResult: application.BatchUserSimpleResult{
			Items: []application.UserSimple{{
				UserID:         42,
				PublicID:       "user_pub_42",
				Nickname:       "Alice",
				AvatarFileID:   "avatar-file-42",
				ProfileVersion: 7,
				Status:         application.UserStatusActive,
			}},
			MissingUserIDs: []application.UserID{404},
		},
	}
	resolver := &fakeAvatarURLResolver{url: "https://cdn.example.com/avatar-file-42.jpg"}
	req := withInternalCaller(
		withJSONHeader(httptest.NewRequest(http.MethodPost, usercontract.BatchSimplePath, bytes.NewBufferString(`{"userIds":[42,404]}`))),
		usercontract.OperationCommentBatchGetAuthorSummaries,
	)
	rr := httptest.NewRecorder()

	NewHandler(service, resolver).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if service.batchSimpleCalls != 1 || len(service.batchSimpleIDs) != 2 || service.batchSimpleIDs[0] != 42 || service.batchSimpleIDs[1] != 404 {
		t.Fatalf("batch simple calls=%d ids=%v", service.batchSimpleCalls, service.batchSimpleIDs)
	}
	var body envelope[usercontract.SimpleBatchResponse]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if len(body.Data.Items) != 1 || body.Data.Items[0].AvatarURL != "https://cdn.example.com/avatar-file-42.jpg" {
		t.Fatalf("simple response = %#v", body.Data)
	}
	if len(body.Data.MissingUserIDs) != 1 || body.Data.MissingUserIDs[0] != 404 {
		t.Fatalf("missing ids = %#v", body.Data.MissingUserIDs)
	}
}

func TestInternalBatchAvailabilityRequiresExpectedOperation(t *testing.T) {
	service := &fakeProfileService{}
	req := withJSONHeader(httptest.NewRequest(http.MethodPost, usercontract.BatchAvailabilityPath, bytes.NewBufferString(`{"userIds":[42]}`)))
	req.Header.Set("X-Caller-Service", "zhicore-comment")
	rr := httptest.NewRecorder()

	NewHandler(service, nil).ServeHTTP(rr, req)

	assertErrorEnvelope(t, rr, http.StatusServiceUnavailable, 1004)
	if service.availabilityCalls != 0 {
		t.Fatalf("availabilityCalls = %d, want 0", service.availabilityCalls)
	}
}

func TestInternalBatchAvailabilityReturnsStatusItems(t *testing.T) {
	service := &fakeProfileService{
		availabilityItems: []application.UserAvailability{
			{UserID: 42, Available: true, Status: application.UserStatusActive},
			{UserID: 77, Available: false, Status: application.UserStatusDeactivated},
		},
	}
	req := withInternalCaller(
		withJSONHeader(httptest.NewRequest(http.MethodPost, usercontract.BatchAvailabilityPath, bytes.NewBufferString(`{"userIds":[42,77]}`))),
		usercontract.OperationCommentCheckUserAvailability,
	)
	rr := httptest.NewRecorder()

	NewHandler(service, nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var body envelope[usercontract.AvailabilityBatchResponse]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if len(body.Data.Items) != 2 || !body.Data.Items[0].Available || body.Data.Items[1].Status != string(application.UserStatusDeactivated) {
		t.Fatalf("availability response = %#v", body.Data)
	}
}

func TestInternalBatchCheckBlockedMapsPairs(t *testing.T) {
	pair := application.UserPair{ActorID: 42, TargetID: 77}
	service := &fakeProfileService{batchBlockedResult: map[application.UserPair]bool{pair: true}}
	req := withInternalCaller(
		withJSONHeader(httptest.NewRequest(http.MethodPost, usercontract.BatchCheckBlockedPath, bytes.NewBufferString(`{"pairs":[{"blockerId":42,"blockedId":77}]}`))),
		usercontract.OperationCommentBatchCheckBlocked,
	)
	rr := httptest.NewRecorder()

	NewHandler(service, nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if service.batchBlockedCalls != 1 || len(service.batchBlockedPairs) != 1 || service.batchBlockedPairs[0] != pair {
		t.Fatalf("blocked calls=%d pairs=%v", service.batchBlockedCalls, service.batchBlockedPairs)
	}
	var body envelope[usercontract.BlockPairsResponse]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if len(body.Data.Items) != 1 || !body.Data.Items[0].Blocked {
		t.Fatalf("blocked response = %#v", body.Data)
	}
}

func TestInternalListFollowerShardMapsRequestAndRequiresNotificationOperation(t *testing.T) {
	service := &fakeProfileService{
		followerShardResult: application.FollowerShardPage{
			FollowerIDs: []application.UserID{101, 99},
			NextCursor:  "99",
			HasMore:     true,
		},
	}
	req := withJSONHeader(httptest.NewRequest(http.MethodPost, usercontract.ListFollowerShardPath, bytes.NewBufferString(`{"followingId":77,"cursor":"42","limit":200}`)))
	req.Header.Set("X-Caller-Service", "zhicore-notification")
	req.Header.Set("X-Caller-Operation", usercontract.OperationNotificationListFollowerShard)
	rr := httptest.NewRecorder()

	NewHandler(service, nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if service.followerShardCalls != 1 {
		t.Fatalf("followerShardCalls = %d, want 1", service.followerShardCalls)
	}
	if service.followerShardQuery.FollowingID != 77 || service.followerShardQuery.Cursor != "42" || service.followerShardQuery.Limit != 200 {
		t.Fatalf("followerShardQuery = %#v", service.followerShardQuery)
	}
	var body envelope[usercontract.ListFollowerShardResponse]
	decodeJSON(t, rr.Body.Bytes(), &body)
	if len(body.Data.FollowerIDs) != 2 || body.Data.FollowerIDs[0] != 101 || body.Data.NextCursor != "99" || !body.Data.HasMore {
		t.Fatalf("follower shard response = %#v", body.Data)
	}
}

func withInternalCaller(req *http.Request, operation string) *http.Request {
	req.Header.Set("X-Caller-Service", "zhicore-comment")
	req.Header.Set("X-Caller-Operation", operation)
	return req
}
