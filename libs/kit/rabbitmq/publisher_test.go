package rabbitmq

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestTopicPublisherPublishesPersistentJSONMessage(t *testing.T) {
	channel := &fakeChannel{deferred: &fakeDeferredConfirmation{ack: true}}
	publisher := NewTopicPublisher(channel, "zhicore.events")
	timestamp := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

	err := publisher.PublishJSON(context.Background(), Message{
		RoutingKey: "comment.liked",
		MessageID:  "evt_comment_liked_1",
		Type:       "comment.liked",
		Timestamp:  timestamp,
		Body:       []byte(`{"eventId":"evt_comment_liked_1"}`),
	})
	if err != nil {
		t.Fatalf("PublishJSON() error = %v", err)
	}

	if channel.exchange != "zhicore.events" || channel.routingKey != "comment.liked" {
		t.Fatalf("publish target = %q/%q", channel.exchange, channel.routingKey)
	}
	if !channel.confirmCalled {
		t.Fatal("Confirm() was not called before publishing")
	}
	if channel.publishing.DeliveryMode != amqp.Persistent {
		t.Fatalf("delivery mode = %d, want persistent", channel.publishing.DeliveryMode)
	}
	if channel.publishing.ContentType != "application/json" {
		t.Fatalf("content type = %q", channel.publishing.ContentType)
	}
	if channel.publishing.MessageId != "evt_comment_liked_1" || channel.publishing.Type != "comment.liked" {
		t.Fatalf("publishing identity = %#v", channel.publishing)
	}
	if !channel.publishing.Timestamp.Equal(timestamp) || string(channel.publishing.Body) != `{"eventId":"evt_comment_liked_1"}` {
		t.Fatalf("publishing body/timestamp = %#v", channel.publishing)
	}
}

func TestTopicPublisherPropagatesPublishError(t *testing.T) {
	channel := &fakeChannel{err: errors.New("broker closed")}
	publisher := NewTopicPublisher(channel, "zhicore.events")

	err := publisher.PublishJSON(context.Background(), Message{
		RoutingKey: "comment.created",
		MessageID:  "evt_1",
		Type:       "comment.created",
		Timestamp:  time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC),
		Body:       []byte(`{}`),
	})
	if !errors.Is(err, channel.err) {
		t.Fatalf("PublishJSON() error = %v, want broker error", err)
	}
}

func TestTopicPublisherReturnsErrorWhenBrokerNacksPublish(t *testing.T) {
	channel := &fakeChannel{deferred: &fakeDeferredConfirmation{ack: false}}
	publisher := NewTopicPublisher(channel, "zhicore.events")

	err := publisher.PublishJSON(context.Background(), Message{
		RoutingKey: "comment.created",
		MessageID:  "evt_1",
		Type:       "comment.created",
		Timestamp:  time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC),
		Body:       []byte(`{}`),
	})
	if err == nil || !strings.Contains(err.Error(), "not acknowledged") {
		t.Fatalf("PublishJSON() error = %v, want nack error", err)
	}
}

func TestTopicPublisherReturnsErrorWhenConfirmWaitExpires(t *testing.T) {
	channel := &fakeChannel{deferred: &fakeDeferredConfirmation{waitForContext: true}}
	publisher := NewTopicPublisher(channel, "zhicore.events", WithPublishConfirmTimeout(time.Nanosecond))

	err := publisher.PublishJSON(context.Background(), Message{
		RoutingKey: "comment.created",
		MessageID:  "evt_1",
		Type:       "comment.created",
		Timestamp:  time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC),
		Body:       []byte(`{}`),
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("PublishJSON() error = %v, want context deadline exceeded", err)
	}
}

type fakeChannel struct {
	exchange      string
	routingKey    string
	publishing    amqp.Publishing
	err           error
	confirmCalled bool
	confirmErr    error
	deferred      DeferredConfirmation
}

func (f *fakeChannel) Confirm(noWait bool) error {
	f.confirmCalled = true
	return f.confirmErr
}

func (f *fakeChannel) PublishWithDeferredConfirmWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (DeferredConfirmation, error) {
	f.exchange = exchange
	f.routingKey = key
	f.publishing = msg
	if f.err != nil {
		return nil, f.err
	}
	if f.deferred == nil {
		f.deferred = &fakeDeferredConfirmation{ack: true}
	}
	return f.deferred, nil
}

type fakeDeferredConfirmation struct {
	ack            bool
	waitForContext bool
	waitCalls      int
	mu             sync.Mutex
}

func (f *fakeDeferredConfirmation) WaitContext(ctx context.Context) (bool, error) {
	f.mu.Lock()
	f.waitCalls++
	f.mu.Unlock()
	if f.waitForContext {
		<-ctx.Done()
		return false, ctx.Err()
	}
	return f.ack, nil
}
