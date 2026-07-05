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
	PublishWithDeferredConfirmWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (DeferredConfirmation, error)
}

type DeferredConfirmation interface {
	WaitContext(context.Context) (bool, error)
}

type AMQPChannel struct {
	channel *amqp.Channel
}

func NewAMQPChannel(channel *amqp.Channel) *AMQPChannel {
	return &AMQPChannel{channel: channel}
}

func (c *AMQPChannel) Confirm(noWait bool) error {
	return c.channel.Confirm(noWait)
}

func (c *AMQPChannel) PublishWithDeferredConfirmWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (DeferredConfirmation, error) {
	return c.channel.PublishWithDeferredConfirmWithContext(ctx, exchange, key, mandatory, immediate, msg)
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
	deferred, err := p.channel.PublishWithDeferredConfirmWithContext(ctx, p.exchange, message.RoutingKey, false, false, publishing)
	if err != nil {
		return fmt.Errorf("publish rabbitmq json message: %w", err)
	}
	if err := p.waitForConfirm(ctx, deferred); err != nil {
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
	p.confirmed = true
	return nil
}

func (p *TopicPublisher) waitForConfirm(ctx context.Context, deferred DeferredConfirmation) error {
	if deferred == nil {
		return fmt.Errorf("wait rabbitmq publish confirm: deferred confirmation is nil")
	}
	// A publish only leaves the outbox after this exact message's deferred
	// confirmation is acked. Nacks and deadline expiry keep the row retryable.
	confirmCtx, cancel := context.WithTimeout(ctx, p.confirmTimeout)
	defer cancel()

	acked, err := deferred.WaitContext(confirmCtx)
	if err != nil {
		return fmt.Errorf("wait rabbitmq publish confirm: %w", err)
	}
	if !acked {
		return fmt.Errorf("wait rabbitmq publish confirm: message not acknowledged")
	}
	return nil
}
