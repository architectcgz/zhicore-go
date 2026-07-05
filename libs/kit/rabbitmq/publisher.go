package rabbitmq

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Channel interface {
	Confirm(noWait bool) error
	NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

const defaultPublishConfirmTimeout = 3 * time.Second

type TopicPublisherOption func(*TopicPublisher)

func WithPublishConfirmTimeout(timeout time.Duration) TopicPublisherOption {
	return func(p *TopicPublisher) {
		if timeout > 0 {
			p.confirmTimeout = timeout
		}
	}
}

type Message struct {
	RoutingKey string
	MessageID  string
	Type       string
	Timestamp  time.Time
	Body       []byte
}

type TopicPublisher struct {
	channel        Channel
	exchange       string
	confirmTimeout time.Duration
	mu             sync.Mutex
	confirmed      bool
	confirms       <-chan amqp.Confirmation
}

func NewTopicPublisher(channel Channel, exchange string, options ...TopicPublisherOption) *TopicPublisher {
	publisher := &TopicPublisher{
		channel:        channel,
		exchange:       strings.TrimSpace(exchange),
		confirmTimeout: defaultPublishConfirmTimeout,
	}
	for _, option := range options {
		if option != nil {
			option(publisher)
		}
	}
	return publisher
}

func (p *TopicPublisher) PublishJSON(ctx context.Context, message Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensureConfirmMode(); err != nil {
		return err
	}

	// Event publishers call this only after an outbox row has been claimed.
	// Persistent delivery makes broker-accepted messages survive broker restarts;
	// the outbox row remains the retry source if publish returns an error.
	publishing := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    message.MessageID,
		Type:         message.Type,
		Timestamp:    message.Timestamp,
		Body:         message.Body,
	}
	if err := p.channel.PublishWithContext(ctx, p.exchange, message.RoutingKey, false, false, publishing); err != nil {
		return fmt.Errorf("publish rabbitmq json message: %w", err)
	}
	if err := p.waitForConfirm(ctx, p.confirms); err != nil {
		return err
	}
	return nil
}

func (p *TopicPublisher) ensureConfirmMode() error {
	if p.confirmed {
		return nil
	}
	if err := p.channel.Confirm(false); err != nil {
		return fmt.Errorf("enable rabbitmq publisher confirms: %w", err)
	}
	p.confirms = p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	p.confirmed = true
	return nil
}

func (p *TopicPublisher) waitForConfirm(ctx context.Context, confirms <-chan amqp.Confirmation) error {
	// A publish only leaves the outbox after the broker explicitly acks it.
	// Nacks, closed confirm streams and deadline expiry keep the row retryable.
	confirmCtx, cancel := context.WithTimeout(ctx, p.confirmTimeout)
	defer cancel()

	select {
	case confirmation, ok := <-confirms:
		if !ok {
			return fmt.Errorf("wait rabbitmq publish confirm: confirm stream closed")
		}
		if !confirmation.Ack {
			return fmt.Errorf("wait rabbitmq publish confirm: message not acknowledged")
		}
		return nil
	case <-confirmCtx.Done():
		return fmt.Errorf("wait rabbitmq publish confirm: %w", confirmCtx.Err())
	}
}
