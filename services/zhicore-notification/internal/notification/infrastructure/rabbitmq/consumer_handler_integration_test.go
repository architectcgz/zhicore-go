package rabbitmq

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	kitrabbitmq "github.com/architectcgz/zhicore-go/libs/kit/rabbitmq"
	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/application"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRabbitMQConsumerHandlerDeadLettersAndAcksIntegration(t *testing.T) {
	url := os.Getenv("ZHICORE_NOTIFICATION_RABBITMQ_INTEGRATION_URL")
	if url == "" {
		t.Skip("set ZHICORE_NOTIFICATION_RABBITMQ_INTEGRATION_URL to run RabbitMQ integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatalf("dial rabbitmq: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("open rabbitmq channel: %v", err)
	}
	t.Cleanup(func() { _ = ch.Close() })

	suffix := time.Now().UnixNano()
	exchange := fmt.Sprintf("zhicore.notification.integration.%d", suffix)
	inputQueue := fmt.Sprintf("zhicore.notification.integration.input.%d", suffix)
	deadQueue := fmt.Sprintf("zhicore.notification.integration.dead.%d", suffix)
	deadRoutingKey := "zhicore-notification.post.dead"
	if err := ch.ExchangeDeclare(exchange, "topic", false, true, false, false, nil); err != nil {
		t.Fatalf("declare exchange: %v", err)
	}
	t.Cleanup(func() {
		cleanupCh, err := conn.Channel()
		if err != nil {
			return
		}
		defer cleanupCh.Close()
		_, _ = cleanupCh.QueueDelete(inputQueue, false, false, false)
		_, _ = cleanupCh.QueueDelete(deadQueue, false, false, false)
		_ = cleanupCh.ExchangeDelete(exchange, false, false)
	})
	if _, err := ch.QueueDeclare(inputQueue, false, false, true, false, nil); err != nil {
		t.Fatalf("declare input queue: %v", err)
	}
	if _, err := ch.QueueDeclare(deadQueue, false, false, true, false, nil); err != nil {
		t.Fatalf("declare dead queue: %v", err)
	}
	if err := ch.QueueBind(inputQueue, "content.post.published", exchange, false, nil); err != nil {
		t.Fatalf("bind input queue: %v", err)
	}
	if err := ch.QueueBind(deadQueue, deadRoutingKey, exchange, false, nil); err != nil {
		t.Fatalf("bind dead queue: %v", err)
	}

	publisher := kitrabbitmq.NewTopicPublisher(kitrabbitmq.NewAMQPChannel(ch), exchange, kitrabbitmq.WithPublishConfirmTimeout(2*time.Second))
	if err := publisher.PublishJSON(ctx, kitrabbitmq.Message{
		RoutingKey: "content.post.published",
		MessageID:  "evt_rabbitmq_integration",
		Type:       "content.post.published",
		Timestamp:  time.Date(2026, 7, 8, 0, 30, 0, 0, time.UTC),
		Body:       []byte(`{"eventId":"evt_rabbitmq_integration"}`),
	}); err != nil {
		t.Fatalf("publish input message: %v", err)
	}

	deliveries, err := ch.Consume(inputQueue, "notification-integration-consumer", false, true, false, false, nil)
	if err != nil {
		t.Fatalf("consume input queue: %v", err)
	}
	var delivery amqp.Delivery
	select {
	case delivery = <-deliveries:
	case <-ctx.Done():
		t.Fatalf("consume input message: %v", ctx.Err())
	}

	app := &fakeInteractionConsumer{result: application.ConsumerResult{
		Action: application.ConsumerActionDeadLetter,
		DeadLetter: &application.DeadLetterEnvelope{
			EventID:     "evt_rabbitmq_integration",
			EventType:   "content.post.published",
			RoutingKey:  "content.post.published",
			Consumer:    "zhicore-notification:post-consumer",
			ErrorClass:  application.ErrorClassProducerContract,
			FailedAt:    time.Date(2026, 7, 8, 0, 31, 0, 0, time.UTC),
			PayloadHash: "integration_hash",
		},
	}}
	handler := NewConsumerHandler(app, publisher, ConsumerHandlerConfig{ConsumerName: "zhicore-notification:post-consumer"})
	if err := handler.Handle(ctx, amqpDelivery{delivery: delivery}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	deadDelivery, ok, err := ch.Get(deadQueue, true)
	if err != nil {
		t.Fatalf("get dead letter message: %v", err)
	}
	if !ok {
		t.Fatal("dead letter queue is empty")
	}
	if deadDelivery.MessageId != "evt_rabbitmq_integration" || deadDelivery.Type != "notification.dead_letter" {
		t.Fatalf("dead letter identity = messageId:%s type:%s", deadDelivery.MessageId, deadDelivery.Type)
	}
	if deadDelivery.RoutingKey != deadRoutingKey {
		t.Fatalf("dead letter routing key = %s, want %s", deadDelivery.RoutingKey, deadRoutingKey)
	}
	if err := ch.Cancel("notification-integration-consumer", false); err != nil {
		t.Fatalf("cancel consumer: %v", err)
	}
	if err := ch.Close(); err != nil {
		t.Fatalf("close rabbitmq channel: %v", err)
	}
	verifyCh, err := conn.Channel()
	if err != nil {
		t.Fatalf("open rabbitmq verification channel: %v", err)
	}
	t.Cleanup(func() { _ = verifyCh.Close() })
	redelivered, ok, err := verifyCh.Get(inputQueue, true)
	if err != nil {
		t.Fatalf("get input queue after ack: %v", err)
	}
	if ok {
		t.Fatalf("input message was redelivered after handler ack: messageId=%s", redelivered.MessageId)
	}
}

type amqpDelivery struct {
	delivery amqp.Delivery
}

func (d amqpDelivery) RoutingKey() string { return d.delivery.RoutingKey }

func (d amqpDelivery) Body() []byte { return d.delivery.Body }

func (d amqpDelivery) RetryCount() int { return 0 }

func (d amqpDelivery) Ack() error { return d.delivery.Ack(false) }

func (d amqpDelivery) Nack(requeue bool) error { return d.delivery.Nack(false, requeue) }
