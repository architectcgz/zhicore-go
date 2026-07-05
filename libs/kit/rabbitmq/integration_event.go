package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const defaultIntegrationPayloadVersion = 1

type JSONPublisher interface {
	PublishJSON(ctx context.Context, message Message) error
}

type IntegrationEvent struct {
	EventID          string
	EventType        string
	PayloadVersion   int
	Producer         string
	OccurredAt       time.Time
	AggregateType    string
	AggregateID      string
	AggregateVersion *int64
	Payload          []byte
}

type IntegrationEventPublisher struct {
	publisher JSONPublisher
	producer  string
}

func NewIntegrationEventPublisher(publisher JSONPublisher, producer string) *IntegrationEventPublisher {
	return &IntegrationEventPublisher{
		publisher: publisher,
		producer:  strings.TrimSpace(producer),
	}
}

func (p *IntegrationEventPublisher) PublishIntegrationEvent(ctx context.Context, event IntegrationEvent) error {
	event.Producer = firstNonEmpty(event.Producer, p.producer)
	payloadVersion := event.PayloadVersion
	if payloadVersion == 0 {
		payloadVersion = defaultIntegrationPayloadVersion
	}

	body, err := json.Marshal(integrationEventEnvelope{
		EventID:          event.EventID,
		EventType:        event.EventType,
		PayloadVersion:   payloadVersion,
		Producer:         event.Producer,
		OccurredAt:       event.OccurredAt.Format(time.RFC3339Nano),
		AggregateType:    event.AggregateType,
		AggregateID:      event.AggregateID,
		AggregateVersion: event.AggregateVersion,
		Payload:          json.RawMessage(event.Payload),
	})
	if err != nil {
		return fmt.Errorf("marshal integration event envelope: %w", err)
	}

	if err := p.publisher.PublishJSON(ctx, Message{
		RoutingKey: event.EventType,
		MessageID:  event.EventID,
		Type:       event.EventType,
		Timestamp:  event.OccurredAt,
		Body:       body,
	}); err != nil {
		// Broker or URL-shaped driver errors can contain userinfo/hosts. The
		// outbox row keeps retry context, so callers only need a safe publish
		// failure classification here.
		return fmt.Errorf("publish integration event: %w", sanitizeIntegrationPublishError(err))
	}
	return nil
}

func sanitizeIntegrationPublishError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return errors.New("rabbitmq publish failed")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type integrationEventEnvelope struct {
	EventID          string          `json:"eventId"`
	EventType        string          `json:"eventType"`
	PayloadVersion   int             `json:"payloadVersion"`
	Producer         string          `json:"producer"`
	OccurredAt       string          `json:"occurredAt"`
	AggregateType    string          `json:"aggregateType"`
	AggregateID      string          `json:"aggregateId"`
	AggregateVersion *int64          `json:"aggregateVersion,omitempty"`
	Payload          json.RawMessage `json:"payload"`
}
