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

func newPostPublishedOutboxEvent(current, published ports.PostRecord, author ports.OwnerSnapshot, publishedBodyID, publishedBodyHash string, publishedAt time.Time) (ports.OutboxEvent, error) {
	payloadJSON, err := json.Marshal(contentevents.PostPublishedPayload{
		PublicID:   current.PublicID,
		InternalID: current.ID,
		AuthorID:   current.OwnerID,
		Author: contentevents.AuthorSnapshot{
			PublicID:    author.PublicID,
			DisplayName: author.DisplayName,
			AvatarURL:   author.AvatarURL,
		},
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

func newPostVisibilityChangedOutboxEvent(current, changed ports.PostRecord, oldVisibility, newVisibility string, publicVisible bool, reason string, changedAt time.Time) (ports.OutboxEvent, error) {
	payloadJSON, err := json.Marshal(contentevents.PostVisibilityChangedPayload{
		PublicID:      current.PublicID,
		InternalID:    current.ID,
		AuthorID:      current.OwnerID,
		OldVisibility: oldVisibility,
		NewVisibility: newVisibility,
		PublicVisible: publicVisible,
		Reason:        reason,
		ChangedAt:     changedAt,
	})
	if err != nil {
		return ports.OutboxEvent{}, err
	}

	return ports.OutboxEvent{
		EventType:        "content.post.visibility_changed",
		PayloadVersion:   1,
		AggregateType:    "post",
		AggregateID:      current.PublicID,
		AggregateVersion: changed.PostVersion,
		PayloadJSON:      payloadJSON,
		OccurredAt:       changedAt,
	}, nil
}

func postVisibilityForStatus(status domain.PostStatus) string {
	switch status {
	case domain.PostStatusPublished:
		return "PUBLIC"
	case domain.PostStatusDeleted:
		return "DELETED"
	case domain.PostStatusDraft, domain.PostStatusScheduled:
		// Scheduled content is not public yet; consumers need visibility
		// semantics, not the author's workflow status.
		return "UNPUBLISHED"
	default:
		return "UNPUBLISHED"
	}
}
