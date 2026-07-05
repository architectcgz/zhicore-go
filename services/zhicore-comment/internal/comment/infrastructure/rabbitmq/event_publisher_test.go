package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	kitrabbitmq "github.com/architectcgz/zhicore-go/libs/kit/rabbitmq"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func TestIntegrationEventPublisherPublishesEnvelopeToTopicExchange(t *testing.T) {
	topic := &fakeTopicPublisher{}
	publisher := NewIntegrationEventPublisher(topic)
	occurredAt := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

	err := publisher.PublishIntegrationEvent(context.Background(), ports.OutboxEvent{
		ID:             42,
		EventID:        "evt_comment_liked_1",
		EventType:      "comment.liked",
		PayloadVersion: 2,
		AggregateType:  "comment",
		AggregateID:    "100",
		Payload:        []byte(`{"commentId":100,"likedBy":200}`),
		OccurredAt:     occurredAt,
	})
	if err != nil {
		t.Fatalf("PublishIntegrationEvent() error = %v", err)
	}

	if topic.message.RoutingKey != "comment.liked" {
		t.Fatalf("routing key = %q", topic.message.RoutingKey)
	}
	if topic.message.MessageID != "evt_comment_liked_1" || topic.message.Type != "comment.liked" || !topic.message.Timestamp.Equal(occurredAt) {
		t.Fatalf("publishing identity = %#v", topic.message)
	}

	var body map[string]any
	if err := json.Unmarshal(topic.message.Body, &body); err != nil {
		t.Fatalf("json.Unmarshal(publishing body) error = %v", err)
	}
	for field, want := range map[string]any{
		"eventId":        "evt_comment_liked_1",
		"eventType":      "comment.liked",
		"payloadVersion": float64(2),
		"producer":       "zhicore-comment",
		"aggregateType":  "comment",
		"aggregateId":    "100",
	} {
		if body[field] != want {
			t.Fatalf("body[%s] = %#v, want %#v; body=%s", field, body[field], want, topic.message.Body)
		}
	}
	if body["occurredAt"] != occurredAt.Format(time.RFC3339Nano) {
		t.Fatalf("occurredAt = %#v", body["occurredAt"])
	}
	payload, ok := body["payload"].(map[string]any)
	if !ok || payload["commentId"] != float64(100) || payload["likedBy"] != float64(200) {
		t.Fatalf("payload = %#v", body["payload"])
	}
}

func TestIntegrationEventPublisherPropagatesPublishError(t *testing.T) {
	topic := &fakeTopicPublisher{err: errors.New("broker closed")}
	publisher := NewIntegrationEventPublisher(topic)

	err := publisher.PublishIntegrationEvent(context.Background(), ports.OutboxEvent{
		EventID:       "evt_1",
		EventType:     "comment.created",
		AggregateType: "comment",
		AggregateID:   "1",
		Payload:       []byte(`{}`),
		OccurredAt:    time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC),
	})
	if err == nil || !errors.Is(err, topic.err) {
		t.Fatalf("PublishIntegrationEvent() error = %v, want broker error", err)
	}
}

type fakeTopicPublisher struct {
	message kitrabbitmq.Message
	err     error
}

func (f *fakeTopicPublisher) PublishJSON(ctx context.Context, message kitrabbitmq.Message) error {
	f.message = message
	return f.err
}
