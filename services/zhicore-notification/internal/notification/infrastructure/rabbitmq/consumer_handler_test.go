package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	kitrabbitmq "github.com/architectcgz/zhicore-go/libs/kit/rabbitmq"
	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/application"
)

func TestConsumerHandlerAcksAfterApplicationAck(t *testing.T) {
	app := &fakeInteractionConsumer{result: application.ConsumerResult{Action: application.ConsumerActionAck}}
	delivery := &fakeDelivery{routingKey: "content.post.liked", body: []byte(`{"eventId":"evt_1"}`)}
	handler := NewConsumerHandler(app, nil, ConsumerHandlerConfig{ConsumerName: "zhicore-notification:content-post-consumer"})

	err := handler.Handle(context.Background(), delivery)

	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if delivery.ackCalls != 1 || delivery.nackCalls != 0 {
		t.Fatalf("ack/nack calls = %d/%d", delivery.ackCalls, delivery.nackCalls)
	}
	if app.events[0].RoutingKey != "content.post.liked" || string(app.events[0].Body) != `{"eventId":"evt_1"}` {
		t.Fatalf("incoming event = %+v", app.events[0])
	}
}

func TestConsumerHandlerNacksTransientFailureWithRequeue(t *testing.T) {
	app := &fakeInteractionConsumer{result: application.ConsumerResult{Action: application.ConsumerActionNackRequeue}}
	delivery := &fakeDelivery{routingKey: "comment.created", body: []byte(`{"eventId":"evt_1"}`)}
	handler := NewConsumerHandler(app, nil, ConsumerHandlerConfig{ConsumerName: "zhicore-notification:comment-consumer"})

	err := handler.Handle(context.Background(), delivery)

	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if delivery.nackCalls != 1 || !delivery.requeue {
		t.Fatalf("nack calls=%d requeue=%t, want requeue", delivery.nackCalls, delivery.requeue)
	}
	if delivery.ackCalls != 0 {
		t.Fatalf("ack calls = %d, want 0", delivery.ackCalls)
	}
}

func TestConsumerHandlerPublishesDeadLetterThenAcks(t *testing.T) {
	app := &fakeInteractionConsumer{result: application.ConsumerResult{
		Action: application.ConsumerActionDeadLetter,
		DeadLetter: &application.DeadLetterEnvelope{
			EventID:     "evt_bad",
			EventType:   "comment.created",
			RoutingKey:  "comment.created",
			Consumer:    "zhicore-notification:comment-consumer",
			ErrorClass:  application.ErrorClassProducerContract,
			FailedAt:    time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC),
			RetryCount:  2,
			PayloadHash: "hash_1",
		},
	}}
	dlq := &fakeDLQPublisher{}
	delivery := &fakeDelivery{routingKey: "comment.created", body: []byte(`{"raw":"payload"}`)}
	handler := NewConsumerHandler(app, dlq, ConsumerHandlerConfig{ConsumerName: "zhicore-notification:comment-consumer"})

	err := handler.Handle(context.Background(), delivery)

	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if delivery.ackCalls != 1 || delivery.nackCalls != 0 {
		t.Fatalf("ack/nack calls = %d/%d", delivery.ackCalls, delivery.nackCalls)
	}
	if dlq.message.RoutingKey != "zhicore-notification.comment.dead" || dlq.message.MessageID != "evt_bad" || dlq.message.Type != "notification.dead_letter" {
		t.Fatalf("dlq message = %+v", dlq.message)
	}
	var body map[string]any
	if err := json.Unmarshal(dlq.message.Body, &body); err != nil {
		t.Fatalf("unmarshal dlq body: %v", err)
	}
	if body["payloadHash"] != "hash_1" || body["eventId"] != "evt_bad" || body["errorClass"] != application.ErrorClassProducerContract {
		t.Fatalf("dlq body = %s", dlq.message.Body)
	}
	if _, exists := body["raw"]; exists {
		t.Fatalf("dlq body leaked raw payload: %s", dlq.message.Body)
	}
}

func TestConsumerHandlerNacksWhenDeadLetterPublishFails(t *testing.T) {
	app := &fakeInteractionConsumer{result: application.ConsumerResult{
		Action:     application.ConsumerActionDeadLetter,
		DeadLetter: &application.DeadLetterEnvelope{EventID: "evt_bad", EventType: "user.followed", RoutingKey: "user.followed", Consumer: "zhicore-notification:user-consumer"},
	}}
	dlq := &fakeDLQPublisher{err: errors.New("broker unavailable")}
	delivery := &fakeDelivery{routingKey: "user.followed", body: []byte(`{"eventId":"evt_bad"}`)}
	handler := NewConsumerHandler(app, dlq, ConsumerHandlerConfig{ConsumerName: "zhicore-notification:user-consumer"})

	err := handler.Handle(context.Background(), delivery)

	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if delivery.nackCalls != 1 || !delivery.requeue || delivery.ackCalls != 0 {
		t.Fatalf("ack/nack/requeue = %d/%d/%t", delivery.ackCalls, delivery.nackCalls, delivery.requeue)
	}
}

type fakeInteractionConsumer struct {
	events []application.IncomingEvent
	result application.ConsumerResult
}

func (f *fakeInteractionConsumer) Handle(ctx context.Context, event application.IncomingEvent) application.ConsumerResult {
	f.events = append(f.events, event)
	return f.result
}

type fakeDelivery struct {
	routingKey string
	body       []byte
	retryCount int
	ackCalls   int
	nackCalls  int
	requeue    bool
}

func (f *fakeDelivery) RoutingKey() string { return f.routingKey }

func (f *fakeDelivery) Body() []byte { return f.body }

func (f *fakeDelivery) RetryCount() int { return f.retryCount }

func (f *fakeDelivery) Ack() error {
	f.ackCalls++
	return nil
}

func (f *fakeDelivery) Nack(requeue bool) error {
	f.nackCalls++
	f.requeue = requeue
	return nil
}

type fakeDLQPublisher struct {
	message kitrabbitmq.Message
	err     error
}

func (f *fakeDLQPublisher) PublishJSON(ctx context.Context, message kitrabbitmq.Message) error {
	f.message = message
	return f.err
}
