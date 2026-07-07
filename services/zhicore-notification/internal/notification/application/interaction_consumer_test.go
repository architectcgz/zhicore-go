package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func TestInteractionConsumerCreatesPostLikedNotification(t *testing.T) {
	deps := newInteractionConsumerDeps()
	consumer := NewInteractionConsumer(InteractionConsumerDeps{
		Notifications: deps.notifications,
		Fanout:        deps.fanout,
		Clock:         deps.clock,
	}, InteractionConsumerConfig{ConsumerName: "zhicore-notification:content-post-consumer", ConsumedEventsRetention: 168 * time.Hour})

	result := consumer.Handle(context.Background(), IncomingEvent{
		RoutingKey: "content.post.liked",
		Body: []byte(`{
			"eventId":"evt_like_1",
			"eventType":"content.post.liked",
			"payloadVersion":1,
			"producer":"zhicore-content",
			"occurredAt":"2026-07-06T10:00:00Z",
			"aggregateType":"post",
			"aggregateId":"post_1",
			"payload":{"publicId":"post_1","internalId":41,"authorId":1001,"likedBy":2002}
		}`),
	})

	if result.Action != ConsumerActionAck {
		t.Fatalf("action = %s, want ack; result=%+v", result.Action, result)
	}
	if len(deps.notifications.created) != 1 {
		t.Fatalf("created notifications = %d, want 1", len(deps.notifications.created))
	}
	created := deps.notifications.created[0]
	if created.Event.EventID != "evt_like_1" || created.Event.ConsumerName != "zhicore-notification:content-post-consumer" {
		t.Fatalf("event metadata = %+v", created.Event)
	}
	if created.RecipientID != 1001 || created.ActorID == nil || *created.ActorID != 2002 {
		t.Fatalf("recipient/actor = %d/%v", created.RecipientID, created.ActorID)
	}
	if created.NotificationType != "POST_LIKED" || created.Category != "INTERACTION" || created.TargetType != "POST" || created.TargetID != "41" {
		t.Fatalf("notification target = %+v", created)
	}
	if created.DedupeKey != "post_liked:41:2002" || created.GroupKey != "post_liked:41" {
		t.Fatalf("dedupe/group = %q/%q", created.DedupeKey, created.GroupKey)
	}
	if len(deps.fanout.hints) != 1 || deps.fanout.hints[0].RecipientID != 1001 {
		t.Fatalf("fanout hints = %+v", deps.fanout.hints)
	}
}

func TestInteractionConsumerPlansPostPublishedCampaignWithoutFollowerFanout(t *testing.T) {
	deps := newInteractionConsumerDeps()
	consumer := newTestInteractionConsumer(deps)

	result := consumer.Handle(context.Background(), IncomingEvent{
		RoutingKey: "content.post.published",
		Body: []byte(`{
			"eventId":"evt_post_published_1",
			"eventType":"content.post.published",
			"payloadVersion":1,
			"producer":"zhicore-content",
			"occurredAt":"2026-07-06T10:00:00Z",
			"aggregateType":"post",
			"aggregateId":"post_41",
			"payload":{"publicId":"post_41","internalId":41,"authorId":1001,"title":"Hello","summary":"Short summary","publishedAt":"2026-07-06T09:59:00Z"}
		}`),
	})

	if result.Action != ConsumerActionAck {
		t.Fatalf("action = %s, want ack; result=%+v", result.Action, result)
	}
	if len(deps.campaigns.planned) != 1 {
		t.Fatalf("planned campaigns = %d, want 1", len(deps.campaigns.planned))
	}
	planned := deps.campaigns.planned[0]
	if planned.Event.EventID != "evt_post_published_1" || planned.AuthorID != 1001 || planned.PostID != 41 {
		t.Fatalf("planned campaign = %+v", planned)
	}
	if planned.Title != "Hello" || planned.Excerpt != "Short summary" || !planned.PublishedAt.Equal(time.Date(2026, 7, 6, 9, 59, 0, 0, time.UTC)) {
		t.Fatalf("planned content snapshot = %+v", planned)
	}
	if planned.AudienceClass != "HOT" || planned.AudienceActiveSince == nil || !planned.AudienceActiveSince.Equal(time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("planned audience = class %q activeSince %v", planned.AudienceClass, planned.AudienceActiveSince)
	}
	if len(deps.notifications.created) != 0 || len(deps.fanout.hints) != 0 {
		t.Fatalf("published campaign must not fanout immediately: notifications=%d hints=%d", len(deps.notifications.created), len(deps.fanout.hints))
	}
}

func TestInteractionConsumerAcksSelfInteractionWithoutWriting(t *testing.T) {
	deps := newInteractionConsumerDeps()
	consumer := newTestInteractionConsumer(deps)

	result := consumer.Handle(context.Background(), IncomingEvent{
		RoutingKey: "user.followed",
		Body: []byte(`{
			"eventId":"evt_follow_self",
			"eventType":"user.followed",
			"payloadVersion":1,
			"producer":"zhicore-user",
			"occurredAt":"2026-07-06T10:00:00Z",
			"aggregateType":"user",
			"aggregateId":"1001",
			"payload":{"followerId":1001,"followingId":1001,"occurredAt":"2026-07-06T10:00:00Z"}
		}`),
	})

	if result.Action != ConsumerActionAck {
		t.Fatalf("action = %s, want ack", result.Action)
	}
	if len(deps.notifications.created) != 0 || len(deps.fanout.hints) != 0 {
		t.Fatalf("self interaction created=%d fanout=%d, want none", len(deps.notifications.created), len(deps.fanout.hints))
	}
}

func TestInteractionConsumerCreatesUserFollowedNotification(t *testing.T) {
	deps := newInteractionConsumerDeps()
	consumer := newTestInteractionConsumer(deps)

	result := consumer.Handle(context.Background(), IncomingEvent{
		RoutingKey: "user.followed",
		Body: []byte(`{
			"eventId":"evt_follow_1",
			"eventType":"user.followed",
			"payloadVersion":1,
			"producer":"zhicore-user",
			"occurredAt":"2026-07-06T10:00:00Z",
			"aggregateType":"user",
			"aggregateId":"1001",
			"payload":{"followerId":2002,"followingId":1001,"occurredAt":"2026-07-06T10:00:00Z"}
		}`),
	})

	if result.Action != ConsumerActionAck {
		t.Fatalf("action = %s, want ack", result.Action)
	}
	if len(deps.notifications.created) != 1 {
		t.Fatalf("created notifications = %d, want 1", len(deps.notifications.created))
	}
	created := deps.notifications.created[0]
	if created.RecipientID != 1001 || created.ActorID == nil || *created.ActorID != 2002 {
		t.Fatalf("recipient/actor = %d/%v", created.RecipientID, created.ActorID)
	}
	if created.NotificationType != "USER_FOLLOWED" || created.Category != "SOCIAL" || created.TargetType != "USER" || created.TargetID != "2002" {
		t.Fatalf("notification = %+v", created)
	}
}

func TestInteractionConsumerCreatesCommentReplyNotificationFromPayloadFacts(t *testing.T) {
	deps := newInteractionConsumerDeps()
	consumer := newTestInteractionConsumer(deps)

	result := consumer.Handle(context.Background(), validCommentReplyEvent())

	if result.Action != ConsumerActionAck {
		t.Fatalf("action = %s, want ack", result.Action)
	}
	if len(deps.notifications.created) != 1 {
		t.Fatalf("created notifications = %d, want 1", len(deps.notifications.created))
	}
	created := deps.notifications.created[0]
	if created.RecipientID != 3003 || created.ActorID == nil || *created.ActorID != 2002 {
		t.Fatalf("recipient/actor = %d/%v", created.RecipientID, created.ActorID)
	}
	if created.NotificationType != "COMMENT_REPLIED" || created.TargetType != "COMMENT" || created.TargetID != "8002" {
		t.Fatalf("notification = %+v", created)
	}
}

func TestInteractionConsumerAcksDuplicateEventNoop(t *testing.T) {
	deps := newInteractionConsumerDeps()
	deps.notifications.err = ports.ErrDuplicateConsumedEvent
	consumer := newTestInteractionConsumer(deps)

	result := consumer.Handle(context.Background(), validCommentReplyEvent())

	if result.Action != ConsumerActionAck {
		t.Fatalf("action = %s, want ack duplicate no-op", result.Action)
	}
	if len(deps.fanout.hints) != 0 {
		t.Fatalf("fanout hints = %+v, want none for duplicate", deps.fanout.hints)
	}
}

func TestInteractionConsumerNacksTransientDependencyFailure(t *testing.T) {
	deps := newInteractionConsumerDeps()
	deps.notifications.err = ports.ErrDependencyUnavailable
	consumer := newTestInteractionConsumer(deps)

	result := consumer.Handle(context.Background(), validCommentReplyEvent())

	if result.Action != ConsumerActionNackRequeue {
		t.Fatalf("action = %s, want nack requeue", result.Action)
	}
}

func TestInteractionConsumerNacksPostPublishedWhenCampaignStoreUnavailable(t *testing.T) {
	deps := newInteractionConsumerDeps()
	deps.campaigns.err = ports.ErrDependencyUnavailable
	consumer := newTestInteractionConsumer(deps)

	result := consumer.Handle(context.Background(), validPostPublishedEvent())

	if result.Action != ConsumerActionNackRequeue {
		t.Fatalf("action = %s, want nack requeue", result.Action)
	}
}

func TestInteractionConsumerDeadLettersInvalidPostPublishedPayload(t *testing.T) {
	deps := newInteractionConsumerDeps()
	consumer := newTestInteractionConsumer(deps)

	result := consumer.Handle(context.Background(), IncomingEvent{
		RoutingKey: "content.post.published",
		Body: []byte(`{
			"eventId":"evt_bad_publish",
			"eventType":"content.post.published",
			"payloadVersion":1,
			"producer":"zhicore-content",
			"occurredAt":"2026-07-06T10:00:00Z",
			"aggregateType":"post",
			"aggregateId":"post_bad",
			"payload":{"publicId":"post_bad","authorId":1001,"title":"Hello","publishedAt":"2026-07-06T09:59:00Z"}
		}`),
	})

	if result.Action != ConsumerActionDeadLetter || result.DeadLetter == nil || result.DeadLetter.ErrorClass != ErrorClassProducerContract {
		t.Fatalf("result = %+v, want producer contract dead-letter", result)
	}
	if len(deps.campaigns.planned) != 0 {
		t.Fatalf("planned campaigns = %d, want none", len(deps.campaigns.planned))
	}
}

func TestInteractionConsumerDeadLettersProducerContractErrorWithoutRawPayload(t *testing.T) {
	deps := newInteractionConsumerDeps()
	consumer := newTestInteractionConsumer(deps)

	result := consumer.Handle(context.Background(), IncomingEvent{
		RoutingKey: "comment.created",
		Body: []byte(`{
			"eventId":"evt_bad_reply",
			"eventType":"comment.created",
			"payloadVersion":1,
			"producer":"zhicore-comment",
			"occurredAt":"2026-07-06T10:00:00Z",
			"aggregateType":"comment",
			"aggregateId":"9001",
			"payload":{"commentId":9001,"publicId":"post_1","internalId":41,"postAuthorId":1001,"authorId":2002,"rootId":8001,"rootAuthorId":3003,"parentId":8002,"hasImages":false,"hasVoice":false,"createdAt":"2026-07-06T10:00:00Z"}
		}`),
	})

	if result.Action != ConsumerActionDeadLetter {
		t.Fatalf("action = %s, want dead-letter; result=%+v", result.Action, result)
	}
	if result.DeadLetter == nil {
		t.Fatal("dead letter = nil")
	}
	if result.DeadLetter.EventID != "evt_bad_reply" ||
		result.DeadLetter.EventType != "comment.created" ||
		result.DeadLetter.RoutingKey != "comment.created" ||
		result.DeadLetter.Consumer != "zhicore-notification:comment-consumer" ||
		result.DeadLetter.ErrorClass != ErrorClassProducerContract ||
		result.DeadLetter.PayloadHash == "" ||
		result.DeadLetter.FailedAt.IsZero() {
		t.Fatalf("dead letter = %+v", result.DeadLetter)
	}
	serialized := result.DeadLetter.String()
	for _, leaked := range []string{"postAuthorId", "parentId", "raw token", "Authorization"} {
		if strings.Contains(serialized, leaked) {
			t.Fatalf("dead letter leaked raw payload content %q: %s", leaked, serialized)
		}
	}
}

func TestInteractionConsumerAcksWhenRealtimeFanoutFails(t *testing.T) {
	deps := newInteractionConsumerDeps()
	deps.fanout.err = errors.New("websocket unavailable")
	consumer := newTestInteractionConsumer(deps)

	result := consumer.Handle(context.Background(), validCommentReplyEvent())

	if result.Action != ConsumerActionAck {
		t.Fatalf("action = %s, want ack despite fanout failure", result.Action)
	}
	if len(deps.notifications.created) != 1 || len(deps.fanout.hints) != 1 {
		t.Fatalf("created=%d fanout=%d", len(deps.notifications.created), len(deps.fanout.hints))
	}
}

func validCommentReplyEvent() IncomingEvent {
	return IncomingEvent{
		RoutingKey: "comment.created",
		Body: []byte(`{
			"eventId":"evt_comment_1",
			"eventType":"comment.created",
			"payloadVersion":1,
			"producer":"zhicore-comment",
			"occurredAt":"2026-07-06T10:00:00Z",
			"aggregateType":"comment",
			"aggregateId":"9001",
			"payload":{"commentId":9001,"publicId":"post_1","internalId":41,"postAuthorId":1001,"authorId":2002,"rootId":8001,"rootAuthorId":3003,"parentId":8002,"parentAuthorId":3003,"hasImages":false,"hasVoice":false,"createdAt":"2026-07-06T10:00:00Z"}
		}`),
	}
}

func validPostPublishedEvent() IncomingEvent {
	return IncomingEvent{
		RoutingKey: "content.post.published",
		Body: []byte(`{
			"eventId":"evt_post_published_1",
			"eventType":"content.post.published",
			"payloadVersion":1,
			"producer":"zhicore-content",
			"occurredAt":"2026-07-06T10:00:00Z",
			"aggregateType":"post",
			"aggregateId":"post_41",
			"payload":{"publicId":"post_41","internalId":41,"authorId":1001,"title":"Hello","summary":"Short summary","publishedAt":"2026-07-06T09:59:00Z"}
		}`),
	}
}

func newTestInteractionConsumer(deps interactionConsumerDeps) *InteractionConsumer {
	return NewInteractionConsumer(InteractionConsumerDeps{
		Notifications: deps.notifications,
		Campaigns:     deps.campaigns,
		Fanout:        deps.fanout,
		Clock:         deps.clock,
	}, InteractionConsumerConfig{ConsumedEventsRetention: 168 * time.Hour})
}

type interactionConsumerDeps struct {
	notifications *fakeInteractionNotificationStore
	campaigns     *fakeCampaignStore
	fanout        *fakeRealtimeFanout
	clock         fakeInteractionClock
}

func newInteractionConsumerDeps() interactionConsumerDeps {
	return interactionConsumerDeps{
		notifications: &fakeInteractionNotificationStore{},
		campaigns:     &fakeCampaignStore{},
		fanout:        &fakeRealtimeFanout{},
		clock:         fakeInteractionClock{now: time.Date(2026, 7, 6, 11, 0, 0, 0, time.UTC)},
	}
}

type fakeInteractionNotificationStore struct {
	created []ports.CreateInteractionNotificationInput
	err     error
}

func (f *fakeInteractionNotificationStore) CreateInteractionNotification(ctx context.Context, input ports.CreateInteractionNotificationInput) (ports.CreateInteractionNotificationResult, error) {
	f.created = append(f.created, input)
	if f.err != nil {
		return ports.CreateInteractionNotificationResult{}, f.err
	}
	return ports.CreateInteractionNotificationResult{Created: true, NotificationID: 10001, PublicID: "ntf_1"}, nil
}

type fakeCampaignStore struct {
	planned           []ports.PlanPostPublishedCampaignInput
	claim             ports.ClaimedCampaignShard
	claimInput        ports.ClaimCampaignShardInput
	failed            []ports.FailCampaignShardInput
	completed         []ports.CompleteCampaignShardInput
	materialized      ports.MaterializeCampaignFollowersResult
	materializeInputs []ports.MaterializeCampaignFollowersInput
	err               error
}

func (f *fakeCampaignStore) PlanPostPublishedCampaign(ctx context.Context, input ports.PlanPostPublishedCampaignInput) (ports.PlanCampaignResult, error) {
	f.planned = append(f.planned, input)
	if f.err != nil {
		return ports.PlanCampaignResult{}, f.err
	}
	return ports.PlanCampaignResult{Created: true, CampaignID: 7001, ShardID: 8001}, nil
}

func (f *fakeCampaignStore) ClaimCampaignShard(_ context.Context, input ports.ClaimCampaignShardInput) (ports.ClaimedCampaignShard, error) {
	f.claimInput = input
	if f.err != nil {
		return ports.ClaimedCampaignShard{}, f.err
	}
	return f.claim, nil
}

func (f *fakeCampaignStore) FailCampaignShard(_ context.Context, input ports.FailCampaignShardInput) error {
	f.failed = append(f.failed, input)
	return nil
}

func (f *fakeCampaignStore) CompleteCampaignShard(_ context.Context, input ports.CompleteCampaignShardInput) error {
	f.completed = append(f.completed, input)
	return f.err
}

func (f *fakeCampaignStore) MaterializeCampaignFollowers(_ context.Context, input ports.MaterializeCampaignFollowersInput) (ports.MaterializeCampaignFollowersResult, error) {
	f.materializeInputs = append(f.materializeInputs, input)
	return f.materialized, f.err
}

func (f *fakeCampaignStore) RebuildGroupState(_ context.Context, input ports.RebuildGroupStateInput) (ports.RebuildGroupStateResult, error) {
	return ports.RebuildGroupStateResult{}, f.err
}

type fakeRealtimeFanout struct {
	hints []RealtimeUnreadHint
	err   error
}

func (f *fakeRealtimeFanout) PublishUnreadHint(ctx context.Context, hint RealtimeUnreadHint) error {
	f.hints = append(f.hints, hint)
	return f.err
}

type fakeInteractionClock struct {
	now time.Time
}

func (c fakeInteractionClock) Now() time.Time {
	return c.now
}
