package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestLikePostIsIdempotentAndAppendsOutboxOnlyOnFirstMutation(t *testing.T) {
	deps := newCreatePostDeps()
	deps.engagement = &fakeEngagementRepository{
		mutateResult: ports.EngagementMutationRecord{
			PostInternalID:   10,
			PostID:           "post_1",
			AuthorID:         7,
			ActorID:          42,
			Changed:          true,
			Liked:            true,
			Favorited:        false,
			AggregateVersion: 3,
			Stats:            ports.PostStatsRecord{LikeCount: 1},
		},
	}
	service := NewService(deps.asDeps())

	got, err := service.LikePost(context.Background(), EngagementCommand{Actor: &Actor{UserID: 42}, PostID: "post_1"})
	if err != nil {
		t.Fatalf("LikePost() error = %v", err)
	}
	if !got.Liked || got.Favorited || got.Stats.LikeCount != 1 {
		t.Fatalf("result = %+v", got)
	}
	if deps.engagement.mutateInput.Action != ports.EngagementActionLike || deps.engagement.mutateInput.ActorID != 42 {
		t.Fatalf("mutation input = %+v", deps.engagement.mutateInput)
	}
	if deps.outbox.appendCalls != 1 {
		t.Fatalf("outbox calls = %d, want 1", deps.outbox.appendCalls)
	}
	if deps.engagementStats.appendCalls != 1 {
		t.Fatalf("engagement stats task calls = %d, want 1", deps.engagementStats.appendCalls)
	}
	if gotTask := deps.engagementStats.appended[0]; gotTask.PostInternalID != 10 || gotTask.PostID != "post_1" || gotTask.Metric != "LIKE" || gotTask.Delta != 1 {
		t.Fatalf("engagement stats task = %+v, want LIKE +1 for post", gotTask)
	}
	var payload map[string]any
	if err := json.Unmarshal(deps.outbox.events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("payload json error = %v", err)
	}
	if deps.outbox.events[0].EventType != "content.post.liked" || payload["likedBy"].(float64) != 42 {
		t.Fatalf("outbox = %+v payload=%s", deps.outbox.events[0], deps.outbox.events[0].PayloadJSON)
	}
	actor, ok := payload["actor"].(map[string]any)
	if !ok || actor["publicId"] != "user_pub_42" || actor["displayName"] != "architect" {
		t.Fatalf("actor snapshot = %#v", payload["actor"])
	}

	deps = newCreatePostDeps()
	deps.engagement = &fakeEngagementRepository{
		mutateResult: ports.EngagementMutationRecord{
			PostInternalID:   10,
			PostID:           "post_1",
			AuthorID:         7,
			ActorID:          42,
			Changed:          false,
			Liked:            true,
			Favorited:        false,
			AggregateVersion: 3,
			Stats:            ports.PostStatsRecord{LikeCount: 1},
		},
	}
	service = NewService(deps.asDeps())

	if _, err := service.LikePost(context.Background(), EngagementCommand{Actor: &Actor{UserID: 42}, PostID: "post_1"}); err != nil {
		t.Fatalf("duplicate LikePost() error = %v", err)
	}
	if deps.outbox.appendCalls != 0 {
		t.Fatalf("duplicate outbox calls = %d, want 0", deps.outbox.appendCalls)
	}
	if deps.engagementStats.appendCalls != 0 {
		t.Fatalf("duplicate stats task calls = %d, want 0", deps.engagementStats.appendCalls)
	}
}

func TestUnlikePostIsIdempotentAndDoesNotEmitDuplicateOutbox(t *testing.T) {
	deps := newCreatePostDeps()
	deps.engagement = &fakeEngagementRepository{
		mutateResult: ports.EngagementMutationRecord{
			PostInternalID:   10,
			PostID:           "post_1",
			AuthorID:         7,
			ActorID:          42,
			Changed:          false,
			Liked:            false,
			Favorited:        true,
			AggregateVersion: 4,
			Stats:            ports.PostStatsRecord{LikeCount: 0, FavoriteCount: 1},
		},
	}
	service := NewService(deps.asDeps())

	got, err := service.UnlikePost(context.Background(), EngagementCommand{Actor: &Actor{UserID: 42}, PostID: "post_1"})
	if err != nil {
		t.Fatalf("UnlikePost() error = %v", err)
	}
	if got.Liked || !got.Favorited || got.Stats.LikeCount != 0 {
		t.Fatalf("result = %+v", got)
	}
	if deps.outbox.appendCalls != 0 {
		t.Fatalf("outbox calls = %d, want 0", deps.outbox.appendCalls)
	}
}

func TestEngagementCommandIgnoresCacheWriteFailureAfterTransaction(t *testing.T) {
	deps := newCreatePostDeps()
	deps.engagement = &fakeEngagementRepository{
		mutateResult: ports.EngagementMutationRecord{
			PostInternalID:   10,
			PostID:           "post_1",
			AuthorID:         7,
			ActorID:          42,
			Changed:          true,
			Liked:            false,
			Favorited:        true,
			AggregateVersion: 5,
			Stats:            ports.PostStatsRecord{FavoriteCount: 1},
		},
	}
	deps.engagementCache = &fakeEngagementCache{writeErr: errors.New("redis unavailable")}
	service := NewService(deps.asDeps())

	got, err := service.FavoritePost(context.Background(), EngagementCommand{Actor: &Actor{UserID: 42}, PostID: "post_1"})
	if err != nil {
		t.Fatalf("FavoritePost() error = %v", err)
	}
	if !got.Favorited || deps.engagementCache.writeCalls != 1 {
		t.Fatalf("result=%+v cacheWrites=%d", got, deps.engagementCache.writeCalls)
	}
}

func TestEngagementWriteRateLimitFailsClosedBeforeMutation(t *testing.T) {
	deps := newCreatePostDeps()
	limiter := &recordingRateLimiter{decision: ports.RateLimitDecision{
		Outcome: ports.RateLimitOutcomeDegradedDenyUnavailable,
		Reason:  "redis_unavailable_fail_closed",
	}}
	serviceDeps := deps.asDeps()
	serviceDeps.Limiter = limiter
	service := NewService(serviceDeps)

	_, err := service.LikePost(context.Background(), EngagementCommand{Actor: &Actor{UserID: 42}, PostID: "post_1"})

	if !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("LikePost() error = %v, want ErrDependencyUnavailable", err)
	}
	if deps.tx.calls != 0 || deps.engagement.mutateCalls != 0 {
		t.Fatalf("side effects tx=%d mutate=%d, want none", deps.tx.calls, deps.engagement.mutateCalls)
	}
	if len(limiter.requests) != 1 {
		t.Fatalf("rate limit requests = %+v, want one", limiter.requests)
	}
	got := limiter.requests[0]
	if got.LimitType != ports.RateLimitTypeEngagementWrite ||
		got.Subject != "actor:42" ||
		got.Resource != "post_1" ||
		got.Operation != "like_post" {
		t.Fatalf("rate limit request = %+v, want engagement write actor/post/operation", got)
	}
}

func TestGetPostEngagementUsesEngagementReadRateLimit(t *testing.T) {
	deps := newCreatePostDeps()
	deps.engagement = &fakeEngagementRepository{
		statsResult: ports.PostEngagementRecord{
			PostID: "post_1",
			Stats:  ports.PostStatsRecord{LikeCount: 3, FavoriteCount: 2},
		},
		statusResult: []ports.EngagementStatusRecord{{PostID: "post_1", Liked: true}},
	}
	limiter := &recordingRateLimiter{decision: ports.RateLimitDecision{Outcome: ports.RateLimitOutcomeAllow}}
	serviceDeps := deps.asDeps()
	serviceDeps.Limiter = limiter
	service := NewService(serviceDeps)

	_, err := service.GetPostEngagement(context.Background(), GetPostEngagementQuery{Actor: &Actor{UserID: 42}, PostID: "post_1"})

	if err != nil {
		t.Fatalf("GetPostEngagement() error = %v", err)
	}
	if len(limiter.requests) != 1 {
		t.Fatalf("rate limit requests = %+v, want one", limiter.requests)
	}
	got := limiter.requests[0]
	if got.LimitType != ports.RateLimitTypeEngagementRead ||
		got.Subject != "actor:42" ||
		got.Resource != "post_1" ||
		got.Operation != "get_post_engagement" {
		t.Fatalf("rate limit request = %+v, want engagement read actor/post/operation", got)
	}
}

func TestBatchGetEngagementStatusRateLimitFailsClosedBeforeCacheOrDB(t *testing.T) {
	deps := newCreatePostDeps()
	limiter := &recordingRateLimiter{decision: ports.RateLimitDecision{
		Outcome: ports.RateLimitOutcomeDegradedDenyUnavailable,
		Reason:  "redis_unavailable_fail_closed",
	}}
	serviceDeps := deps.asDeps()
	serviceDeps.Limiter = limiter
	service := NewService(serviceDeps)

	_, err := service.BatchGetEngagementStatus(context.Background(), BatchGetEngagementStatusQuery{
		Actor:   &Actor{UserID: 42},
		PostIDs: []string{"post_1", "post_2"},
	})

	if !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("BatchGetEngagementStatus() error = %v, want ErrDependencyUnavailable", err)
	}
	if deps.engagementCache.readCalls != 0 || deps.engagement.statusCalls != 0 {
		t.Fatalf("status side effects cache=%d db=%d, want none", deps.engagementCache.readCalls, deps.engagement.statusCalls)
	}
	if len(limiter.requests) != 1 {
		t.Fatalf("rate limit requests = %+v, want one", limiter.requests)
	}
	got := limiter.requests[0]
	if got.LimitType != ports.RateLimitTypeEngagementRead ||
		got.Subject != "actor:42" ||
		got.Resource != "post_1,post_2" ||
		got.Operation != "batch_get_engagement_status" {
		t.Fatalf("rate limit request = %+v, want engagement read actor/batch/operation", got)
	}
}

func TestGetPostEngagementReturnsUnknownViewerWhenFallbackCannotConfirm(t *testing.T) {
	deps := newCreatePostDeps()
	deps.engagement = &fakeEngagementRepository{
		statsResult: ports.PostEngagementRecord{
			PostID: "post_1",
			Stats:  ports.PostStatsRecord{LikeCount: 3, FavoriteCount: 2},
		},
		statusErr: errors.New("db fallback exhausted"),
	}
	deps.engagementCache = &fakeEngagementCache{readErr: errors.New("redis unavailable")}
	service := NewService(deps.asDeps())

	got, err := service.GetPostEngagement(context.Background(), GetPostEngagementQuery{Actor: &Actor{UserID: 42}, PostID: "post_1"})
	if err != nil {
		t.Fatalf("GetPostEngagement() error = %v", err)
	}
	if got.Viewer == nil || got.Viewer.Liked.Ptr() != nil || got.Viewer.Favorited.Ptr() != nil || !got.Viewer.Degraded {
		t.Fatalf("viewer = %+v, want unknown degraded", got.Viewer)
	}
	if got.Stats.LikeCount != 3 || got.Stats.FavoriteCount != 2 {
		t.Fatalf("stats = %+v", got.Stats)
	}
}

func TestBatchGetEngagementStatusDedupesAndUsesSingleBatchRepositoryCall(t *testing.T) {
	deps := newCreatePostDeps()
	deps.engagement = &fakeEngagementRepository{
		statusResult: []ports.EngagementStatusRecord{
			{PostID: "post_1", Liked: true, Favorited: false},
			{PostID: "post_2", Liked: false, Favorited: true},
		},
	}
	deps.engagementCache = &fakeEngagementCache{readErr: errors.New("redis unavailable")}
	service := NewService(deps.asDeps())

	got, err := service.BatchGetEngagementStatus(context.Background(), BatchGetEngagementStatusQuery{
		Actor:   &Actor{UserID: 42},
		PostIDs: []string{"post_1", "post_2", "post_1"},
	})
	if err != nil {
		t.Fatalf("BatchGetEngagementStatus() error = %v", err)
	}
	if deps.engagement.statusCalls != 1 {
		t.Fatalf("status calls = %d, want one batch call", deps.engagement.statusCalls)
	}
	if len(deps.engagement.statusPostIDs) != 2 || deps.engagement.statusPostIDs[0] != "post_1" || deps.engagement.statusPostIDs[1] != "post_2" {
		t.Fatalf("status post ids = %#v", deps.engagement.statusPostIDs)
	}
	if len(got.Items) != 2 || !got.Items[0].Liked.value || !got.Items[1].Favorited.value {
		t.Fatalf("items = %+v", got.Items)
	}
}

func TestEngagementMapsRepositoryErrors(t *testing.T) {
	deps := newCreatePostDeps()
	deps.engagement = &fakeEngagementRepository{mutateErr: domain.ErrPostNotFound}
	service := NewService(deps.asDeps())

	_, err := service.LikePost(context.Background(), EngagementCommand{Actor: &Actor{UserID: 42}, PostID: "post_missing"})

	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("LikePost() error = %v, want ErrPostNotFound", err)
	}
}
