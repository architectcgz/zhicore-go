package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

const (
	ErrorClassProducerContract = "PRODUCER_CONTRACT"
	ErrorClassDependency       = "DEPENDENCY_UNAVAILABLE"

	ConsumerActionAck         ConsumerAction = "ack"
	ConsumerActionNackRequeue ConsumerAction = "nack_requeue"
	ConsumerActionDeadLetter  ConsumerAction = "dead_letter"
)

type ConsumerAction string

type IncomingEvent struct {
	RoutingKey string
	Body       []byte
	RetryCount int
}

type ConsumerResult struct {
	Action     ConsumerAction
	DeadLetter *DeadLetterEnvelope
}

type DeadLetterEnvelope struct {
	EventID     string    `json:"eventId"`
	EventType   string    `json:"eventType"`
	RoutingKey  string    `json:"routingKey"`
	Consumer    string    `json:"consumer"`
	ErrorClass  string    `json:"errorClass"`
	FailedAt    time.Time `json:"failedAt"`
	RetryCount  int       `json:"retryCount"`
	PayloadHash string    `json:"payloadHash"`
}

func (e DeadLetterEnvelope) String() string {
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintf("dead-letter eventId=%s eventType=%s", e.EventID, e.EventType)
	}
	return string(body)
}

type RealtimeUnreadHint struct {
	RecipientID    int64
	NotificationID int64
	PublicID       string
	EventType      string
}

type RealtimeFanoutPublisher interface {
	PublishUnreadHint(ctx context.Context, hint RealtimeUnreadHint) error
}

type InteractionConsumerDeps struct {
	Notifications ports.InteractionNotificationStore
	Fanout        RealtimeFanoutPublisher
	Clock         ports.Clock
}

type InteractionConsumerConfig struct {
	ConsumerName            string
	ConsumedEventsRetention time.Duration
}

type InteractionConsumer struct {
	notifications ports.InteractionNotificationStore
	fanout        RealtimeFanoutPublisher
	clock         ports.Clock
	config        InteractionConsumerConfig
}

func NewInteractionConsumer(deps InteractionConsumerDeps, config InteractionConsumerConfig) *InteractionConsumer {
	if deps.Clock == nil {
		deps.Clock = systemClock{}
	}
	if config.ConsumedEventsRetention <= 0 {
		config.ConsumedEventsRetention = 7 * 24 * time.Hour
	}
	return &InteractionConsumer{
		notifications: deps.Notifications,
		fanout:        deps.Fanout,
		clock:         deps.Clock,
		config:        config,
	}
}

func (c *InteractionConsumer) Handle(ctx context.Context, event IncomingEvent) ConsumerResult {
	envelope, err := decodeIntegrationEnvelope(event.Body)
	consumerName := c.consumerName(event.RoutingKey)
	payloadHash := hashPayload(event.Body)
	if err != nil {
		return c.deadLetter(event, envelope, consumerName, payloadHash, ErrorClassProducerContract)
	}

	input, noop, err := c.notificationInput(envelope, event, consumerName, payloadHash)
	if err != nil {
		return c.deadLetter(event, envelope, consumerName, payloadHash, ErrorClassProducerContract)
	}
	if noop {
		return ConsumerResult{Action: ConsumerActionAck}
	}
	if c.notifications == nil {
		return ConsumerResult{Action: ConsumerActionNackRequeue}
	}

	result, err := c.notifications.CreateInteractionNotification(ctx, input)
	if errors.Is(err, ports.ErrDuplicateConsumedEvent) {
		return ConsumerResult{Action: ConsumerActionAck}
	}
	if err != nil {
		return ConsumerResult{Action: ConsumerActionNackRequeue}
	}
	if result.Created && c.fanout != nil {
		// Realtime fanout is a best-effort hint after the inbox write. It must
		// not roll back or requeue the authoritative in-app notification.
		_ = c.fanout.PublishUnreadHint(ctx, RealtimeUnreadHint{
			RecipientID:    input.RecipientID,
			NotificationID: result.NotificationID,
			PublicID:       result.PublicID,
			EventType:      envelope.EventType,
		})
	}
	return ConsumerResult{Action: ConsumerActionAck}
}

func (c *InteractionConsumer) notificationInput(envelope integrationEnvelope, event IncomingEvent, consumerName string, payloadHash string) (ports.CreateInteractionNotificationInput, bool, error) {
	occurredAt, err := time.Parse(time.RFC3339Nano, envelope.OccurredAt)
	if err != nil {
		return ports.CreateInteractionNotificationInput{}, false, fmt.Errorf("occurredAt is invalid")
	}
	base := ports.CreateInteractionNotificationInput{
		Event: ports.ConsumedEventMetadata{
			EventID:      envelope.EventID,
			EventType:    envelope.EventType,
			RoutingKey:   firstNonEmpty(event.RoutingKey, envelope.EventType),
			ConsumerName: consumerName,
			PayloadHash:  payloadHash,
			OccurredAt:   occurredAt,
			ExpiresAt:    c.clock.Now().Add(c.config.ConsumedEventsRetention),
		},
		Category:      "INTERACTION",
		EventCode:     envelope.EventType,
		Importance:    "NORMAL",
		SourceEventID: envelope.EventID,
		Payload:       envelope.Payload,
		OccurredAt:    occurredAt,
		CreatedAt:     c.clock.Now(),
	}

	switch envelope.EventType {
	case "content.post.liked":
		return postLikedNotification(base, envelope.Payload)
	case "comment.created":
		return commentCreatedNotification(base, envelope.Payload)
	case "user.followed":
		return userFollowedNotification(base, envelope.Payload)
	default:
		return ports.CreateInteractionNotificationInput{}, true, nil
	}
}

func postLikedNotification(base ports.CreateInteractionNotificationInput, raw json.RawMessage) (ports.CreateInteractionNotificationInput, bool, error) {
	var payload struct {
		InternalID int64 `json:"internalId"`
		AuthorID   int64 `json:"authorId"`
		LikedBy    int64 `json:"likedBy"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ports.CreateInteractionNotificationInput{}, false, err
	}
	if payload.InternalID <= 0 || payload.AuthorID <= 0 || payload.LikedBy <= 0 {
		return ports.CreateInteractionNotificationInput{}, false, fmt.Errorf("content.post.liked payload is incomplete")
	}
	if payload.AuthorID == payload.LikedBy {
		return ports.CreateInteractionNotificationInput{}, true, nil
	}
	actorID := payload.LikedBy
	base.RecipientID = payload.AuthorID
	base.ActorID = &actorID
	base.NotificationType = "POST_LIKED"
	base.TargetType = "POST"
	base.TargetID = strconv.FormatInt(payload.InternalID, 10)
	base.DedupeKey = fmt.Sprintf("post_liked:%d:%d", payload.InternalID, payload.LikedBy)
	base.GroupKey = fmt.Sprintf("post_liked:%d", payload.InternalID)
	base.Title = "New like"
	base.Content = "liked your post"
	return base, false, nil
}

func commentCreatedNotification(base ports.CreateInteractionNotificationInput, raw json.RawMessage) (ports.CreateInteractionNotificationInput, bool, error) {
	var payload struct {
		CommentID      int64 `json:"commentId"`
		InternalID     int64 `json:"internalId"`
		PostAuthorID   int64 `json:"postAuthorId"`
		AuthorID       int64 `json:"authorId"`
		RootID         int64 `json:"rootId"`
		RootAuthorID   int64 `json:"rootAuthorId"`
		ParentID       int64 `json:"parentId"`
		ParentAuthorID int64 `json:"parentAuthorId"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ports.CreateInteractionNotificationInput{}, false, err
	}
	if payload.CommentID <= 0 || payload.InternalID <= 0 || payload.PostAuthorID <= 0 || payload.AuthorID <= 0 {
		return ports.CreateInteractionNotificationInput{}, false, fmt.Errorf("comment.created payload is incomplete")
	}
	if payload.ParentID > 0 || payload.RootID > 0 {
		if payload.ParentID <= 0 || payload.ParentAuthorID <= 0 || payload.RootID <= 0 || payload.RootAuthorID <= 0 {
			return ports.CreateInteractionNotificationInput{}, false, fmt.Errorf("comment.created reply payload is incomplete")
		}
		if payload.ParentAuthorID == payload.AuthorID {
			return ports.CreateInteractionNotificationInput{}, true, nil
		}
		actorID := payload.AuthorID
		base.RecipientID = payload.ParentAuthorID
		base.ActorID = &actorID
		base.NotificationType = "COMMENT_REPLIED"
		base.TargetType = "COMMENT"
		base.TargetID = strconv.FormatInt(payload.ParentID, 10)
		base.DedupeKey = fmt.Sprintf("comment_replied:%d:%d", payload.ParentID, payload.CommentID)
		base.GroupKey = fmt.Sprintf("comment_replied:%d", payload.ParentID)
		base.Title = "New reply"
		base.Content = "replied to your comment"
		return base, false, nil
	}
	if payload.PostAuthorID == payload.AuthorID {
		return ports.CreateInteractionNotificationInput{}, true, nil
	}
	actorID := payload.AuthorID
	base.RecipientID = payload.PostAuthorID
	base.ActorID = &actorID
	base.NotificationType = "POST_COMMENTED"
	base.TargetType = "POST"
	base.TargetID = strconv.FormatInt(payload.InternalID, 10)
	base.DedupeKey = fmt.Sprintf("post_commented:%d:%d", payload.InternalID, payload.CommentID)
	base.GroupKey = fmt.Sprintf("post_commented:%d", payload.InternalID)
	base.Title = "New comment"
	base.Content = "commented on your post"
	return base, false, nil
}

func userFollowedNotification(base ports.CreateInteractionNotificationInput, raw json.RawMessage) (ports.CreateInteractionNotificationInput, bool, error) {
	var payload struct {
		FollowerID  int64 `json:"followerId"`
		FollowingID int64 `json:"followingId"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ports.CreateInteractionNotificationInput{}, false, err
	}
	if payload.FollowerID <= 0 || payload.FollowingID <= 0 {
		return ports.CreateInteractionNotificationInput{}, false, fmt.Errorf("user.followed payload is incomplete")
	}
	if payload.FollowerID == payload.FollowingID {
		return ports.CreateInteractionNotificationInput{}, true, nil
	}
	actorID := payload.FollowerID
	base.Category = "SOCIAL"
	base.RecipientID = payload.FollowingID
	base.ActorID = &actorID
	base.NotificationType = "USER_FOLLOWED"
	base.TargetType = "USER"
	base.TargetID = strconv.FormatInt(payload.FollowerID, 10)
	base.DedupeKey = fmt.Sprintf("user_followed:%d:%d", payload.FollowingID, payload.FollowerID)
	base.GroupKey = fmt.Sprintf("user_followed:%d", payload.FollowerID)
	base.Title = "New follower"
	base.Content = "started following you"
	return base, false, nil
}

func (c *InteractionConsumer) deadLetter(event IncomingEvent, envelope integrationEnvelope, consumerName, payloadHash, errorClass string) ConsumerResult {
	return ConsumerResult{
		Action: ConsumerActionDeadLetter,
		DeadLetter: &DeadLetterEnvelope{
			EventID:     firstNonEmpty(envelope.EventID, "unknown"),
			EventType:   firstNonEmpty(envelope.EventType, event.RoutingKey, "unknown"),
			RoutingKey:  firstNonEmpty(event.RoutingKey, envelope.EventType, "unknown"),
			Consumer:    consumerName,
			ErrorClass:  errorClass,
			FailedAt:    c.clock.Now(),
			RetryCount:  event.RetryCount,
			PayloadHash: payloadHash,
		},
	}
}

func (c *InteractionConsumer) consumerName(routingKey string) string {
	if strings.TrimSpace(c.config.ConsumerName) != "" {
		return strings.TrimSpace(c.config.ConsumerName)
	}
	switch {
	case strings.HasPrefix(routingKey, "content.post."):
		return "zhicore-notification:content-post-consumer"
	case strings.HasPrefix(routingKey, "comment."):
		return "zhicore-notification:comment-consumer"
	case strings.HasPrefix(routingKey, "user."):
		return "zhicore-notification:user-consumer"
	default:
		return "zhicore-notification:interaction-consumer"
	}
}

type integrationEnvelope struct {
	EventID        string          `json:"eventId"`
	EventType      string          `json:"eventType"`
	PayloadVersion int             `json:"payloadVersion"`
	Producer       string          `json:"producer"`
	OccurredAt     string          `json:"occurredAt"`
	AggregateType  string          `json:"aggregateType"`
	AggregateID    string          `json:"aggregateId"`
	Payload        json.RawMessage `json:"payload"`
}

func decodeIntegrationEnvelope(body []byte) (integrationEnvelope, error) {
	var envelope integrationEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return integrationEnvelope{}, err
	}
	if strings.TrimSpace(envelope.EventID) == "" ||
		strings.TrimSpace(envelope.EventType) == "" ||
		envelope.PayloadVersion <= 0 ||
		strings.TrimSpace(envelope.Producer) == "" ||
		strings.TrimSpace(envelope.OccurredAt) == "" ||
		len(envelope.Payload) == 0 {
		return envelope, fmt.Errorf("integration event envelope is incomplete")
	}
	return envelope, nil
}

func hashPayload(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
