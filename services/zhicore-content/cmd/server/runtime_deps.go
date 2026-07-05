package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	contentbody "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/body"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
	contentruntime "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/runtime"
	_ "github.com/lib/pq"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type openedContentRuntime struct {
	Module    *contentruntime.Module
	Readiness ReadinessController
	Workers   WorkerLifecycle
	Closers   []namedCloser
}

func openContentRuntimeDependencies(ctx context.Context, cfg ContentServerConfig) (openedContentRuntime, error) {
	postgresDB, err := sql.Open("postgres", cfg.Postgres.DSN)
	if err != nil {
		return openedContentRuntime{}, fmt.Errorf("open postgres dependency: %w", err)
	}
	closers := []namedCloser{{name: "postgres", closer: postgresDB}}
	if err := postgresDB.PingContext(ctx); err != nil {
		closeNamedClosers(closers)
		return openedContentRuntime{}, fmt.Errorf("ping postgres dependency: %w", err)
	}

	mongoClient, err := drivermongo.Connect(options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		closeNamedClosers(closers)
		return openedContentRuntime{}, fmt.Errorf("open mongo dependency: %w", err)
	}
	closers = append(closers, namedCloser{name: "mongo", closer: closeFunc(func() error {
		return mongoClient.Disconnect(context.Background())
	})})
	if err := mongoClient.Ping(ctx, readpref.Primary()); err != nil {
		closeNamedClosers(closers)
		return openedContentRuntime{}, fmt.Errorf("ping mongo dependency: %w", err)
	}

	readiness := newReadinessSwitch()
	rabbitmq := unavailableHealthChecker{component: "rabbitmq publisher"}
	module, err := contentruntime.Build(contentruntime.Deps{
		Config: &contentruntime.Config{
			ServiceName: cfg.ServiceName,
			Workers: contentruntime.WorkerConfig{
				CleanupEnabled: cfg.Workers.CleanupEnabled,
				RepairEnabled:  cfg.Workers.RepairEnabled,
				OutboxEnabled:  cfg.Workers.OutboxEnabled,
			},
		},
		PostgresDB:     postgresDB,
		BodyCollection: mongoClient.Database(cfg.Mongo.Database).Collection(cfg.Mongo.BodyCollection),
		Health: contentruntime.HealthCheckers{
			Lifecycle: readiness,
			Postgres:  postgresPingChecker{db: postgresDB},
			Mongo:     mongoPingChecker{client: mongoClient},
			RabbitMQ:  rabbitmq,
		},
		Parser:            contentbody.NewV1BodyParser(contentbody.DefaultBodyValidationPolicy()),
		Outbox:            unavailableOutboxPublisher{},
		IntegrationEvents: unavailableIntegrationEventPublisher{},
		Clock:             systemClock{},
		Users:             unavailableUserProfileClient{},
		Files:             unavailableFileResourceClient{},
	})
	if err != nil {
		closeNamedClosers(closers)
		return openedContentRuntime{}, fmt.Errorf("build content runtime module: %w", err)
	}

	return openedContentRuntime{
		Module:    module,
		Readiness: readiness,
		Workers:   newContentWorkerLifecycle(module.Workers),
		Closers:   closers,
	}, nil
}

type readinessSwitch struct {
	ready atomic.Bool
}

func newReadinessSwitch() *readinessSwitch {
	return &readinessSwitch{}
}

func (r *readinessSwitch) MarkReady() {
	r.ready.Store(true)
}

func (r *readinessSwitch) MarkNotReady() {
	r.ready.Store(false)
}

func (r *readinessSwitch) IsReady() bool {
	return r.ready.Load()
}

func (r *readinessSwitch) Check(context.Context) error {
	if r.IsReady() {
		return nil
	}
	return errors.New("content server is not ready")
}

type postgresPingChecker struct {
	db *sql.DB
}

func (c postgresPingChecker) Check(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

type mongoPingChecker struct {
	client *drivermongo.Client
}

func (c mongoPingChecker) Check(ctx context.Context) error {
	return c.client.Ping(ctx, readpref.Primary())
}

type unavailableHealthChecker struct {
	component string
}

func (c unavailableHealthChecker) Check(context.Context) error {
	return fmt.Errorf("%s is not implemented", c.component)
}

type noopWorkerLifecycle struct{}

func (noopWorkerLifecycle) Start(context.Context) error { return nil }

func (noopWorkerLifecycle) StopAcceptingNewWork() {}

func (noopWorkerLifecycle) Wait(context.Context) error { return nil }

type contentWorkerLifecycle struct {
	runners []contentruntime.Worker
	cancel  context.CancelFunc
	done    chan error
}

func newContentWorkerLifecycle(descriptors []contentruntime.WorkerDescriptor) WorkerLifecycle {
	runners := make([]contentruntime.Worker, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Enabled && descriptor.Runner != nil {
			runners = append(runners, descriptor.Runner)
		}
	}
	if len(runners) == 0 {
		return noopWorkerLifecycle{}
	}
	return &contentWorkerLifecycle{runners: runners}
}

func (l *contentWorkerLifecycle) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if l.done != nil {
		return nil
	}
	workerCtx, cancel := context.WithCancel(ctx)
	l.cancel = cancel
	l.done = make(chan error, len(l.runners))
	for _, runner := range l.runners {
		runner := runner
		go func() {
			l.done <- runner.Run(workerCtx)
		}()
	}
	return nil
}

func (l *contentWorkerLifecycle) StopAcceptingNewWork() {
	if l.cancel != nil {
		l.cancel()
	}
}

func (l *contentWorkerLifecycle) Wait(ctx context.Context) error {
	if l.done == nil {
		return nil
	}
	var errs []error
	for range l.runners {
		select {
		case err := <-l.done:
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				errs = append(errs, err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return errors.Join(errs...)
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

type unavailableOutboxPublisher struct{}

func (unavailableOutboxPublisher) Append(context.Context, ports.Tx, ports.OutboxEvent) error {
	return application.ErrDependencyUnavailable
}

type unavailableIntegrationEventPublisher struct{}

func (unavailableIntegrationEventPublisher) PublishIntegrationEvent(context.Context, ports.OutboxEvent) error {
	return application.ErrDependencyUnavailable
}

type unavailableUserProfileClient struct{}

func (unavailableUserProfileClient) GetOwnerSnapshot(context.Context, int64) (ports.OwnerSnapshot, error) {
	return ports.OwnerSnapshot{}, application.ErrDependencyUnavailable
}

type unavailableFileResourceClient struct{}

func (unavailableFileResourceClient) ValidateBodyMediaRefs(context.Context, []ports.MediaRef) error {
	return application.ErrDependencyUnavailable
}

func (unavailableFileResourceClient) ValidateCoverFile(context.Context, string) error {
	return application.ErrDependencyUnavailable
}

func closeNamedClosers(closers []namedCloser) {
	for _, closer := range closers {
		if closer.closer == nil {
			continue
		}
		_ = closer.closer.Close()
	}
}
