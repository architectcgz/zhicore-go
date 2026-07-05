package runtime

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	kitrabbitmq "github.com/architectcgz/zhicore-go/libs/kit/rabbitmq"
	commenthttp "github.com/architectcgz/zhicore-go/services/zhicore-comment/api/http"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/application"
	commentpostgres "github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/infrastructure/postgres"
	commentrabbitmq "github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/infrastructure/rabbitmq"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
	"github.com/gin-gonic/gin"
)

type Worker interface {
	Run(context.Context) error
}

type Deps struct {
	Service         commenthttp.Service
	Workers         []Worker
	PostgresDB      *sql.DB
	RabbitMQChannel kitrabbitmq.Channel
	Clock           ports.Clock
	Outbox          OutboxConfig
}

type Module struct {
	HTTPHandler *gin.Engine
	Workers     []Worker
}

type OutboxConfig struct {
	Enabled      bool
	DispatcherID string
	Exchange     string
	BatchSize    int
	MaxAttempts  int
	RetryBackoff time.Duration
	StaleAfter   time.Duration
	PollInterval time.Duration
}

const defaultEventExchange = "zhicore.events"

func Build(deps Deps) (*Module, error) {
	if deps.Service == nil {
		return nil, fmt.Errorf("comment runtime Service dependency is required")
	}

	workers := append([]Worker(nil), deps.Workers...)
	if deps.Outbox.Enabled {
		outboxWorker, err := buildOutboxWorker(deps)
		if err != nil {
			return nil, err
		}
		workers = append(workers, outboxWorker)
	}

	root := commenthttp.NewHandler(deps.Service)
	root.GET("/health/live", healthHandler())
	root.GET("/health/ready", healthHandler())

	return &Module{
		HTTPHandler: root,
		Workers:     workers,
	}, nil
}

func buildOutboxWorker(deps Deps) (Worker, error) {
	if deps.PostgresDB == nil {
		return nil, fmt.Errorf("comment runtime PostgresDB dependency is required when outbox is enabled")
	}
	if deps.RabbitMQChannel == nil {
		return nil, fmt.Errorf("comment runtime RabbitMQChannel dependency is required when outbox is enabled")
	}
	if strings.TrimSpace(deps.Outbox.DispatcherID) == "" {
		return nil, fmt.Errorf("comment runtime Outbox.DispatcherID is required when outbox is enabled")
	}
	clock := deps.Clock
	if clock == nil {
		clock = systemClock{}
	}
	exchange := strings.TrimSpace(deps.Outbox.Exchange)
	if exchange == "" {
		exchange = defaultEventExchange
	}

	repository := commentpostgres.NewOutboxDispatchRepository(deps.PostgresDB)
	topicPublisher := kitrabbitmq.NewTopicPublisher(deps.RabbitMQChannel, exchange)
	eventPublisher := commentrabbitmq.NewIntegrationEventPublisher(topicPublisher)
	dispatcher, err := application.NewOutboxDispatcher(application.OutboxDispatcherConfig{
		DispatcherID: deps.Outbox.DispatcherID,
		BatchSize:    deps.Outbox.BatchSize,
		MaxAttempts:  deps.Outbox.MaxAttempts,
		RetryBackoff: deps.Outbox.RetryBackoff,
		StaleAfter:   deps.Outbox.StaleAfter,
		Repository:   repository,
		Publisher:    eventPublisher,
		Clock:        clock,
	})
	if err != nil {
		return nil, fmt.Errorf("build comment outbox dispatcher: %w", err)
	}
	return outboxWorker{dispatcher: dispatcher, interval: deps.Outbox.PollInterval}, nil
}

type outboxWorker struct {
	dispatcher *application.OutboxDispatcher
	interval   time.Duration
}

func (w outboxWorker) Run(ctx context.Context) error {
	return w.dispatcher.Run(ctx, w.interval)
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

func healthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Readiness stays dependency-free until PostgreSQL, RabbitMQ, Redis and
		// downstream clients have concrete runtime adapters wired.
		c.String(http.StatusOK, "ok")
	}
}
