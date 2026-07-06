package runtime

import (
	"database/sql"
	"time"

	kitrabbitmq "github.com/architectcgz/zhicore-go/libs/kit/rabbitmq"
	contentbody "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/body"
	contentclients "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/clients"
	contentpostgres "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/postgres"
	contentrabbitmq "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/rabbitmq"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
	amqp "github.com/rabbitmq/amqp091-go"
)

func NewDefaultBodyParser() ports.BodyParserRegistry {
	return contentbody.NewV1BodyParser(contentbody.DefaultBodyValidationPolicy())
}

func NewPostgresOutboxPublisher(db *sql.DB) ports.OutboxPublisher {
	return contentpostgres.NewStore(db, contentpostgres.StoreConfig{})
}

func NewRabbitMQIntegrationEventPublisher(channel *amqp.Channel, exchange string, confirmTimeout time.Duration) ports.IntegrationEventPublisher {
	topicPublisher := kitrabbitmq.NewTopicPublisher(
		kitrabbitmq.NewAMQPChannel(channel),
		exchange,
		kitrabbitmq.WithPublishConfirmTimeout(confirmTimeout),
	)
	return contentrabbitmq.NewIntegrationEventPublisher(topicPublisher)
}

func NewUserProfileClient(baseURL string, timeout time.Duration) ports.UserProfileClient {
	return contentclients.NewUserClient(contentclients.UserClientConfig{
		BaseURL: baseURL,
		Timeout: timeout,
	})
}

func NewFileResourceClient(baseURL string, timeout time.Duration, maxAttempts int) ports.FileResourceClient {
	return contentclients.NewFileClient(contentclients.FileClientConfig{
		BaseURL:     baseURL,
		Timeout:     timeout,
		MaxAttempts: maxAttempts,
	})
}
