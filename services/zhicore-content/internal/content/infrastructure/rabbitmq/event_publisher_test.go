package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	kitrabbitmq "github.com/architectcgz/zhicore-go/libs/kit/rabbitmq"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestIntegrationEventPublisherPublishesContentEnvelopeToTopicExchange(t *testing.T) {
	topic := &fakeTopicPublisher{}
	publisher := NewIntegrationEventPublisher(topic)
	occurredAt := time.Date(2026, 7, 5, 15, 30, 0, 0, time.UTC)

	err := publisher.PublishIntegrationEvent(context.Background(), ports.OutboxEvent{
		ID:               42,
		EventID:          "evt_post_published_1",
		EventType:        "content.post.published",
		PayloadVersion:   2,
		AggregateType:    "post",
		AggregateID:      "post_1",
		AggregateVersion: 7,
		PayloadJSON:      []byte(`{"postId":"post_1","publishedBy":1001}`),
		OccurredAt:       occurredAt,
	})
	if err != nil {
		t.Fatalf("PublishIntegrationEvent() error = %v", err)
	}

	if topic.message.RoutingKey != "content.post.published" {
		t.Fatalf("routing key = %q", topic.message.RoutingKey)
	}
	if topic.message.MessageID != "evt_post_published_1" || topic.message.Type != "content.post.published" || !topic.message.Timestamp.Equal(occurredAt) {
		t.Fatalf("publishing identity = %#v", topic.message)
	}

	var body map[string]any
	if err := json.Unmarshal(topic.message.Body, &body); err != nil {
		t.Fatalf("json.Unmarshal(publishing body) error = %v", err)
	}
	for field, want := range map[string]any{
		"eventId":          "evt_post_published_1",
		"eventType":        "content.post.published",
		"payloadVersion":   float64(2),
		"producer":         "zhicore-content",
		"aggregateType":    "post",
		"aggregateId":      "post_1",
		"aggregateVersion": float64(7),
	} {
		if body[field] != want {
			t.Fatalf("body[%s] = %#v, want %#v; body=%s", field, body[field], want, topic.message.Body)
		}
	}
	if body["occurredAt"] != occurredAt.Format(time.RFC3339Nano) {
		t.Fatalf("occurredAt = %#v", body["occurredAt"])
	}
	payload, ok := body["payload"].(map[string]any)
	if !ok || payload["postId"] != "post_1" || payload["publishedBy"] != float64(1001) {
		t.Fatalf("payload = %#v", body["payload"])
	}
}

func TestIntegrationEventPublisherRedactsBrokerError(t *testing.T) {
	topic := &fakeTopicPublisher{err: errors.New("amqp://content:secret@mq.internal:5672 closed")}
	publisher := NewIntegrationEventPublisher(topic)

	err := publisher.PublishIntegrationEvent(context.Background(), ports.OutboxEvent{
		EventID:       "evt_1",
		EventType:     "content.post.published",
		AggregateType: "post",
		AggregateID:   "post_1",
		PayloadJSON:   []byte(`{}`),
		OccurredAt:    time.Date(2026, 7, 5, 15, 30, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("PublishIntegrationEvent() error = nil, want broker error")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "content:secret@") || strings.Contains(err.Error(), "mq.internal") {
		t.Fatalf("PublishIntegrationEvent() leaked broker error: %v", err)
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
