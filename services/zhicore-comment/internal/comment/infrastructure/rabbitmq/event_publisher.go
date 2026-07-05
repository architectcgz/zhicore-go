package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kitrabbitmq "github.com/architectcgz/zhicore-go/libs/kit/rabbitmq"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

const (
	commentProducer       = "zhicore-comment"
	defaultPayloadVersion = 1
)

type TopicPublisher interface {
	PublishJSON(ctx context.Context, message kitrabbitmq.Message) error
}

type IntegrationEventPublisher struct {
	publisher TopicPublisher
}

func NewIntegrationEventPublisher(publisher TopicPublisher) *IntegrationEventPublisher {
	return &IntegrationEventPublisher{publisher: publisher}
}

func (p *IntegrationEventPublisher) PublishIntegrationEvent(ctx context.Context, event ports.OutboxEvent) error {
	payloadVersion := event.PayloadVersion
	if payloadVersion == 0 {
		payloadVersion = defaultPayloadVersion
	}
	body, err := json.Marshal(integrationEventEnvelope{
		EventID:        event.EventID,
		EventType:      event.EventType,
		PayloadVersion: payloadVersion,
		Producer:       commentProducer,
		OccurredAt:     event.OccurredAt.Format(time.RFC3339Nano),
		AggregateType:  event.AggregateType,
		AggregateID:    event.AggregateID,
		Payload:        json.RawMessage(event.Payload),
	})
	if err != nil {
		return fmt.Errorf("marshal comment integration event envelope: %w", err)
	}

	if err := p.publisher.PublishJSON(ctx, kitrabbitmq.Message{
		RoutingKey: event.EventType,
		MessageID:  event.EventID,
		Type:       event.EventType,
		Timestamp:  event.OccurredAt,
		Body:       body,
	}); err != nil {
		return fmt.Errorf("publish comment integration event: %w", err)
	}
	return nil
}

type integrationEventEnvelope struct {
	EventID        string          `json:"eventId"`
	EventType      string          `json:"eventType"`
	PayloadVersion int             `json:"payloadVersion"`
	Producer       string          `json:"producer"`
	OccurredAt     string          `json:"occurredAt"`
	AggregateType  string          `json:"aggregateType"`
	AggregateID    string          `json:"aggregateId"`
	Payload        json.RawMessage `json:"payload"`
}
