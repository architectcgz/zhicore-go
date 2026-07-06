package application

import (
	"encoding/json"
	"time"

	contentevents "github.com/architectcgz/zhicore-go/libs/contracts/events/content"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func hasPostPublishedEvent(events []domain.DomainEvent) bool {
	for _, event := range events {
		if _, ok := event.(domain.PostPublished); ok {
			return true
		}
	}
	return false
}

// Application owns the mapping from domain publish facts to the cross-service
// outbox contract so the domain model stays free of MQ / JSON concerns.

func newPostPublishedOutboxEvent(current, published ports.PostRecord, publishedBodyID, publishedBodyHash string, publishedAt time.Time) (ports.OutboxEvent, error) {
	payloadJSON, err := json.Marshal(contentevents.PostPublishedPayload{
		PublicID:          current.PublicID,
		InternalID:        current.ID,
		AuthorID:          current.OwnerID,
		Title:             current.DraftTitle,
		Summary:           current.DraftSummary,
		CoverFileID:       current.DraftCoverFileID,
		PublishedAt:       publishedAt,
		PublishedBodyID:   publishedBodyID,
		PublishedBodyHash: publishedBodyHash,
	})
	if err != nil {
		return ports.OutboxEvent{}, err
	}

	return ports.OutboxEvent{
		EventType:        "content.post.published",
		PayloadVersion:   1,
		AggregateType:    "post",
		AggregateID:      current.PublicID,
		AggregateVersion: published.PostVersion,
		PayloadJSON:      payloadJSON,
		OccurredAt:       publishedAt,
	}, nil
}
