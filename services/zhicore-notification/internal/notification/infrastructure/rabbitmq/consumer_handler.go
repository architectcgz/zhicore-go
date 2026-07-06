package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	kitrabbitmq "github.com/architectcgz/zhicore-go/libs/kit/rabbitmq"
	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/application"
)

type InteractionConsumer interface {
	Handle(context.Context, application.IncomingEvent) application.ConsumerResult
}

type Delivery interface {
	RoutingKey() string
	Body() []byte
	RetryCount() int
	Ack() error
	Nack(requeue bool) error
}

type ConsumerHandlerConfig struct {
	ConsumerName string
}

type ConsumerHandler struct {
	consumer InteractionConsumer
	dlq      kitrabbitmq.JSONPublisher
	config   ConsumerHandlerConfig
}

func NewConsumerHandler(consumer InteractionConsumer, dlq kitrabbitmq.JSONPublisher, config ConsumerHandlerConfig) *ConsumerHandler {
	return &ConsumerHandler{consumer: consumer, dlq: dlq, config: config}
}

func (h *ConsumerHandler) Handle(ctx context.Context, delivery Delivery) error {
	if h.consumer == nil {
		return fmt.Errorf("notification interaction consumer is required")
	}
	result := h.consumer.Handle(ctx, application.IncomingEvent{
		RoutingKey: delivery.RoutingKey(),
		Body:       delivery.Body(),
		RetryCount: delivery.RetryCount(),
	})
	switch result.Action {
	case application.ConsumerActionAck:
		return delivery.Ack()
	case application.ConsumerActionNackRequeue:
		return delivery.Nack(true)
	case application.ConsumerActionDeadLetter:
		if err := h.publishDeadLetter(ctx, result.DeadLetter); err != nil {
			return delivery.Nack(true)
		}
		return delivery.Ack()
	default:
		return delivery.Nack(true)
	}
}

func (h *ConsumerHandler) publishDeadLetter(ctx context.Context, envelope *application.DeadLetterEnvelope) error {
	if envelope == nil {
		return fmt.Errorf("dead letter envelope is required")
	}
	if h.dlq == nil {
		return fmt.Errorf("dead letter publisher is required")
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal notification dead letter: %w", err)
	}
	return h.dlq.PublishJSON(ctx, kitrabbitmq.Message{
		RoutingKey: deadLetterRoutingKey(firstNonEmpty(envelope.Consumer, h.config.ConsumerName)),
		MessageID:  envelope.EventID,
		Type:       "notification.dead_letter",
		Timestamp:  envelope.FailedAt,
		Body:       body,
	})
}

func deadLetterRoutingKey(consumerName string) string {
	normalized := strings.TrimSpace(consumerName)
	normalized = strings.TrimPrefix(normalized, "zhicore-notification:")
	normalized = strings.TrimSuffix(normalized, "-consumer")
	if normalized == "" {
		normalized = "interaction"
	}
	return "zhicore-notification." + normalized + ".dead"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
