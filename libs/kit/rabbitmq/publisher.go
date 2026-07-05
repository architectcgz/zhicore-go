package rabbitmq

import (
	"context"
	"fmt"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Channel interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

type Message struct {
	RoutingKey string
	MessageID  string
	Type       string
	Timestamp  time.Time
	Body       []byte
}

type TopicPublisher struct {
	channel  Channel
	exchange string
}

func NewTopicPublisher(channel Channel, exchange string) *TopicPublisher {
	return &TopicPublisher{
		channel:  channel,
		exchange: strings.TrimSpace(exchange),
	}
}

func (p *TopicPublisher) PublishJSON(ctx context.Context, message Message) error {
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
	return nil
}
