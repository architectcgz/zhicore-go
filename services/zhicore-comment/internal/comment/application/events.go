package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	commentevents "github.com/architectcgz/zhicore-go/libs/contracts/events/comment"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func (s *Service) publishCreated(ctx context.Context, event domain.CommentCreated, post ports.CommentablePost, occurredAt time.Time) error {
	comment := event.CreatedComment()
	payload := commentevents.CommentCreatedPayload{
		CommentID:    int64(comment.ID),
		PublicID:     string(post.PostID),
		InternalID:   int64(post.ContentInternalID),
		PostAuthorID: int64(post.AuthorID),
		AuthorID:     int64(comment.AuthorID),
		HasImages:    len(comment.Media.ImageFileIDs) > 0,
		HasVoice:     strings.TrimSpace(comment.Media.VoiceFileID) != "",
		CreatedAt:    occurredAt.UTC().Format(time.RFC3339),
	}
	if root, ok := event.RootComment(); ok {
		parent, _ := event.ParentComment()
		payload.RootID = int64Ptr(int64(root.ID))
		payload.RootAuthorID = int64Ptr(int64(root.AuthorID))
		payload.ParentID = int64Ptr(int64(parent.ID))
		payload.ParentAuthorID = int64Ptr(int64(parent.AuthorID))
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal comment created event: %w", err)
	}
	if err := s.outbox.Publish(ctx, ports.OutboxMessage{
		EventType:     commentCreatedEventType,
		AggregateType: "comment",
		AggregateID:   strconv.FormatInt(int64(comment.ID), 10),
		OccurredAt:    occurredAt,
		Payload:       body,
	}); err != nil {
		return fmt.Errorf("publish comment created outbox: %w", err)
	}
	return nil
}

func (s *Service) publishDeleted(ctx context.Context, deleted ports.DeleteSubtreeResult, deletedBy domain.UserID, role DeletedByRole, reason string, occurredAt time.Time) error {
	entry := deleted.Entry
	payload := commentevents.CommentDeletedPayload{
		CommentID:     int64(entry.ID),
		PublicID:      string(entry.PostID),
		InternalID:    int64(entry.ContentInternalID),
		AuthorID:      int64(entry.AuthorID),
		DeletedBy:     int64(deletedBy),
		DeletedByRole: string(role),
		DeletedAt:     occurredAt.UTC().Format(time.RFC3339),
		IsRoot:        entry.IsTopLevel(),
		AffectedCount: deleted.AffectedCount,
	}
	if entry.IsReply() {
		payload.RootID = int64Ptr(int64(entry.RootID))
	}
	if strings.TrimSpace(reason) != "" {
		payload.DeleteReason = stringPtr(strings.TrimSpace(reason))
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal comment deleted event: %w", err)
	}
	return s.outbox.Publish(ctx, ports.OutboxMessage{
		EventType:     commentDeletedEventType,
		AggregateType: "comment",
		AggregateID:   strconv.FormatInt(int64(entry.ID), 10),
		OccurredAt:    occurredAt,
		Payload:       body,
	})
}

func (s *Service) publishLikeChanged(ctx context.Context, eventType string, comment domain.Comment, actorID domain.UserID, occurredAt time.Time) error {
	var payload any
	if eventType == commentLikedEventType {
		payload = commentevents.CommentLikedPayload{
			CommentID:       int64(comment.ID),
			PublicID:        string(comment.PostID),
			InternalID:      int64(comment.ContentInternalID),
			CommentAuthorID: int64(comment.AuthorID),
			LikedBy:         int64(actorID),
			OccurredAt:      occurredAt.UTC().Format(time.RFC3339),
		}
	} else {
		payload = commentevents.CommentUnlikedPayload{
			CommentID:       int64(comment.ID),
			PublicID:        string(comment.PostID),
			InternalID:      int64(comment.ContentInternalID),
			CommentAuthorID: int64(comment.AuthorID),
			UnlikedBy:       int64(actorID),
			OccurredAt:      occurredAt.UTC().Format(time.RFC3339),
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s event: %w", eventType, err)
	}
	return s.outbox.Publish(ctx, ports.OutboxMessage{
		EventType:     eventType,
		AggregateType: "comment",
		AggregateID:   strconv.FormatInt(int64(comment.ID), 10),
		OccurredAt:    occurredAt,
		Payload:       body,
	})
}

func int64Ptr(value int64) *int64 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
