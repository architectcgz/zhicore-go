package rabbitmq

import (
	"context"

	kitrabbitmq "github.com/architectcgz/zhicore-go/libs/kit/rabbitmq"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const contentProducer = "zhicore-content"

type IntegrationEventPublisher struct {
	publisher *kitrabbitmq.IntegrationEventPublisher
}

func NewIntegrationEventPublisher(publisher kitrabbitmq.JSONPublisher) *IntegrationEventPublisher {
	return &IntegrationEventPublisher{
		publisher: kitrabbitmq.NewIntegrationEventPublisher(publisher, contentProducer),
	}
}

func (p *IntegrationEventPublisher) PublishIntegrationEvent(ctx context.Context, event ports.OutboxEvent) error {
	var aggregateVersion *int64
	if event.AggregateVersion != 0 {
		aggregateVersion = &event.AggregateVersion
	}
	return p.publisher.PublishIntegrationEvent(ctx, kitrabbitmq.IntegrationEvent{
		EventID:          event.EventID,
		EventType:        event.EventType,
		PayloadVersion:   event.PayloadVersion,
		AggregateType:    event.AggregateType,
		AggregateID:      event.AggregateID,
		AggregateVersion: aggregateVersion,
		Payload:          event.PayloadJSON,
		OccurredAt:       event.OccurredAt,
	})
}
