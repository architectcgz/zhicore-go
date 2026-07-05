package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestIntegrationEventPublisherPublishesEnvelope(t *testing.T) {
	topic := &fakeJSONPublisher{}
	publisher := NewIntegrationEventPublisher(topic, "zhicore-content")
	occurredAt := time.Date(2026, 7, 5, 16, 0, 0, 0, time.UTC)
	version := int64(7)

	err := publisher.PublishIntegrationEvent(context.Background(), IntegrationEvent{
		EventID:          "evt_post_published_1",
		EventType:        "content.post.published",
		PayloadVersion:   2,
		AggregateType:    "post",
		AggregateID:      "post_1",
		AggregateVersion: &version,
		Payload:          []byte(`{"postId":"post_1"}`),
		OccurredAt:       occurredAt,
	})
	if err != nil {
		t.Fatalf("PublishIntegrationEvent() error = %v", err)
	}
	if topic.message.RoutingKey != "content.post.published" ||
		topic.message.MessageID != "evt_post_published_1" ||
		topic.message.Type != "content.post.published" ||
		!topic.message.Timestamp.Equal(occurredAt) {
		t.Fatalf("message = %#v, want content.post.published identity", topic.message)
	}

	var body map[string]any
	if err := json.Unmarshal(topic.message.Body, &body); err != nil {
		t.Fatalf("json.Unmarshal(body) error = %v", err)
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
		t.Fatalf("occurredAt = %#v, want RFC3339Nano", body["occurredAt"])
	}
	if payload := body["payload"].(map[string]any); payload["postId"] != "post_1" {
		t.Fatalf("payload = %#v, want post_1", payload)
	}
}

func TestIntegrationEventPublisherOmitsMissingAggregateVersion(t *testing.T) {
	topic := &fakeJSONPublisher{}
	publisher := NewIntegrationEventPublisher(topic, "zhicore-comment")

	err := publisher.PublishIntegrationEvent(context.Background(), IntegrationEvent{
		EventID:       "evt_comment_created_1",
		EventType:     "comment.created",
		AggregateType: "comment",
		AggregateID:   "comment_1",
		Payload:       []byte(`{"commentId":"comment_1"}`),
		OccurredAt:    time.Date(2026, 7, 5, 16, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("PublishIntegrationEvent() error = %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(topic.message.Body, &body); err != nil {
		t.Fatalf("json.Unmarshal(body) error = %v", err)
	}
	if _, exists := body["aggregateVersion"]; exists {
		t.Fatalf("aggregateVersion = %#v, want omitted without real version", body["aggregateVersion"])
	}
	if body["payloadVersion"] != float64(1) {
		t.Fatalf("payloadVersion = %#v, want default 1", body["payloadVersion"])
	}
}

func TestIntegrationEventPublisherRedactsPublishErrors(t *testing.T) {
	topic := &fakeJSONPublisher{err: errors.New("amqp://content:secret@mq.internal:5672 closed")}
	publisher := NewIntegrationEventPublisher(topic, "zhicore-content")

	err := publisher.PublishIntegrationEvent(context.Background(), IntegrationEvent{
		EventID:       "evt_1",
		EventType:     "content.post.published",
		AggregateType: "post",
		AggregateID:   "post_1",
		Payload:       []byte(`{}`),
		OccurredAt:    time.Date(2026, 7, 5, 16, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("PublishIntegrationEvent() error = nil, want publish error")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "content:secret@") || strings.Contains(err.Error(), "mq.internal") {
		t.Fatalf("PublishIntegrationEvent() leaked publish error: %v", err)
	}
}

type fakeJSONPublisher struct {
	message Message
	err     error
}

func (f *fakeJSONPublisher) PublishJSON(ctx context.Context, message Message) error {
	f.message = message
	return f.err
}
