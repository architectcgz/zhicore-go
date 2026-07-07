package runtime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	contenthttp "github.com/architectcgz/zhicore-go/services/zhicore-content/api/http"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
	contentmongo "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/mongo"
	contentpostgres "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/postgres"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
	"github.com/gin-gonic/gin"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
)

type Config struct {
	ServiceName string
	Workers     WorkerConfig
}

type WorkerConfig struct {
	CleanupEnabled         bool
	RepairEnabled          bool
	OutboxEnabled          bool
	EngagementStatsEnabled bool
	PollInterval           time.Duration
}

type HealthChecker interface {
	Check(context.Context) error
}

type HealthCheckers struct {
	Lifecycle HealthChecker
	Postgres  HealthChecker
	Mongo     HealthChecker
	RabbitMQ  HealthChecker
}

type Deps struct {
	Config            *Config
	PostgresDB        *sql.DB
	BodyCollection    *drivermongo.Collection
	Health            HealthCheckers
	Workers           []WorkerDescriptor
	Parser            ports.BodyParserRegistry
	Outbox            ports.OutboxPublisher
	IntegrationEvents ports.IntegrationEventPublisher
	Clock             ports.Clock
	Users             ports.UserProfileClient
	Files             ports.FileResourceClient
}

type Module struct {
	HTTPHandler   *gin.Engine
	Workers       []WorkerDescriptor
	HealthDetails HealthDetails
}

type Worker interface {
	Run(context.Context) error
}

type WorkerDescriptor struct {
	Name           string        `json:"name"`
	Enabled        bool          `json:"enabled"`
	DisabledReason string        `json:"disabledReason,omitempty"`
	Checker        HealthChecker `json:"-"`
	Runner         Worker        `json:"-"`
}

type HealthDetails struct {
	Service    string             `json:"service"`
	Postgres   string             `json:"postgres"`
	Mongo      string             `json:"mongo"`
	BodyParser string             `json:"bodyParser"`
	Workers    []WorkerDescriptor `json:"workers"`
}

func Build(deps Deps) (*Module, error) {
	if err := validateDeps(deps); err != nil {
		return nil, err
	}

	store := contentpostgres.NewStore(deps.PostgresDB, contentpostgres.StoreConfig{})
	bodyStore := contentmongo.NewBodyStore(deps.BodyCollection, nil)
	cleanupStore := contentpostgres.NewCleanupTaskStore(store)
	repairStore := contentpostgres.NewRepairTaskStore(store)
	engagementStatsStore := contentpostgres.NewEngagementStatsTaskStore(store)
	outboxDispatch := contentpostgres.NewOutboxDispatchRepository(deps.PostgresDB)
	outboxAdmin := contentpostgres.NewOutboxAdminRepository(deps.PostgresDB)
	service := application.NewService(application.Deps{
		Posts:           store,
		Queries:         store,
		Bodies:          bodyStore,
		Cleanup:         cleanupStore,
		Repair:          repairStore,
		Outbox:          deps.Outbox,
		Admin:           outboxAdmin,
		Taxonomy:        store,
		Engagement:      store,
		EngagementStats: engagementStatsStore,
		Users:           deps.Users,
		Files:           deps.Files,
		Tx:              contentpostgres.NewTransactionRunner(deps.PostgresDB),
		Parser:          deps.Parser,
		Clock:           deps.Clock,
	})

	workers := configuredWorkerDescriptors(deps.Workers, deps.Config.Workers, cleanupStore, repairStore, engagementStatsStore, outboxDispatch, deps.IntegrationEvents, bodyStore, store, deps.Clock)
	health := HealthDetails{
		Service:    serviceName(deps.Config),
		Postgres:   "configured",
		Mongo:      "configured",
		BodyParser: "v1",
		Workers:    workers,
	}
	root := contenthttp.NewHandler(service)
	root.GET("/health/live", healthHandler(health))
	root.GET("/health/ready", readyHandler(health, deps.Health, workers))

	return &Module{
		HTTPHandler:   root,
		Workers:       workers,
		HealthDetails: health,
	}, nil
}

func validateDeps(deps Deps) error {
	if deps.Config == nil {
		return fmt.Errorf("content runtime Config dependency is required")
	}
	if deps.PostgresDB == nil {
		return fmt.Errorf("content runtime PostgresDB dependency is required")
	}
	if deps.BodyCollection == nil {
		return fmt.Errorf("content runtime BodyCollection dependency is required")
	}
	if deps.Health.Postgres == nil {
		return fmt.Errorf("content runtime Postgres health checker dependency is required")
	}
	if deps.Health.Mongo == nil {
		return fmt.Errorf("content runtime Mongo health checker dependency is required")
	}
	if deps.Health.RabbitMQ == nil {
		return fmt.Errorf("content runtime RabbitMQ health checker dependency is required")
	}
	if deps.Parser == nil {
		return fmt.Errorf("content runtime Parser dependency is required")
	}
	if deps.Outbox == nil {
		return fmt.Errorf("content runtime Outbox dependency is required")
	}
	if deps.Config.Workers.OutboxEnabled && deps.IntegrationEvents == nil {
		return fmt.Errorf("content runtime IntegrationEvents dependency is required when outbox worker is enabled")
	}
	if deps.Clock == nil {
		return fmt.Errorf("content runtime Clock dependency is required")
	}
	if deps.Users == nil {
		return fmt.Errorf("content runtime Users dependency is required")
	}
	if deps.Files == nil {
		return fmt.Errorf("content runtime Files dependency is required")
	}
	return nil
}

func serviceName(config *Config) string {
	if config == nil || strings.TrimSpace(config.ServiceName) == "" {
		return "zhicore-content"
	}
	return strings.TrimSpace(config.ServiceName)
}

func configuredWorkerDescriptors(
	extra []WorkerDescriptor,
	config WorkerConfig,
	cleanupStore ports.BodyCleanupTaskStore,
	repairStore ports.BodyRepairTaskStore,
	engagementStatsStore ports.EngagementStatsTaskStore,
	outboxDispatch ports.OutboxDispatchRepository,
	integrationEvents ports.IntegrationEventPublisher,
	bodyStore ports.PostContentStore,
	references ports.BodyReferenceChecker,
	clock ports.Clock,
) []WorkerDescriptor {
	workers := configuredContentWorkers(config, cleanupStore, repairStore, engagementStatsStore, outboxDispatch, integrationEvents, bodyStore, references, clock)
	if len(extra) == 0 {
		return workers
	}
	copied := make([]WorkerDescriptor, 0, len(workers)+len(extra))
	copied = append(copied, workers...)
	copied = append(copied, extra...)
	return copied
}

func configuredContentWorkers(
	config WorkerConfig,
	cleanupStore ports.BodyCleanupTaskStore,
	repairStore ports.BodyRepairTaskStore,
	engagementStatsStore ports.EngagementStatsTaskStore,
	outboxDispatch ports.OutboxDispatchRepository,
	integrationEvents ports.IntegrationEventPublisher,
	bodyStore ports.PostContentStore,
	references ports.BodyReferenceChecker,
	clock ports.Clock,
) []WorkerDescriptor {
	workers := []WorkerDescriptor{
		disabledWorkerDescriptor("content-body-cleanup", "disabled by configuration"),
		disabledWorkerDescriptor("content-body-repair", "disabled by configuration"),
		disabledWorkerDescriptor("content-engagement-stats", "disabled by configuration"),
		disabledWorkerDescriptor("content-outbox-dispatcher", "disabled until outbox dispatcher is implemented"),
	}
	interval := config.PollInterval
	if interval <= 0 {
		interval = time.Minute
	}
	if config.CleanupEnabled {
		cleanup := application.NewBodyCleanupWorker(application.BodyCleanupWorkerDeps{
			Tasks:      cleanupStore,
			Bodies:     bodyStore,
			References: references,
			Clock:      clock,
		}, application.BodyCleanupWorkerConfig{
			WorkerID:        "content-body-cleanup",
			BatchSize:       100,
			StaleClaimAfter: 5 * time.Minute,
			RetryBackoff:    time.Minute,
			DeadThreshold:   3,
		})
		workers[0] = enabledWorkerDescriptor("content-body-cleanup", pollingWorker{name: "content-body-cleanup", interval: interval, runUntilIdle: cleanup.RunUntilIdle})
	}
	if config.RepairEnabled {
		repair := application.NewBodyRepairWorker(application.BodyRepairWorkerDeps{
			Tasks: repairStore,
			Clock: clock,
		}, application.BodyRepairWorkerConfig{
			WorkerID:        "content-body-repair",
			BatchSize:       100,
			StaleClaimAfter: 5 * time.Minute,
			RetryBackoff:    time.Minute,
			DeadThreshold:   1,
		})
		workers[1] = enabledWorkerDescriptor("content-body-repair", pollingWorker{name: "content-body-repair", interval: interval, runUntilIdle: repair.RunUntilIdle})
	}
	if config.EngagementStatsEnabled {
		engagementStats := application.NewEngagementStatsWorker(application.EngagementStatsWorkerDeps{
			Tasks: engagementStatsStore,
			Clock: clock,
		}, application.EngagementStatsWorkerConfig{
			WorkerID:        "content-engagement-stats",
			BatchSize:       100,
			StaleClaimAfter: 5 * time.Minute,
			RetryBackoff:    time.Minute,
			DeadThreshold:   5,
		})
		workers[2] = enabledWorkerDescriptor("content-engagement-stats", pollingWorker{name: "content-engagement-stats", interval: interval, runUntilIdle: engagementStats.RunUntilIdle})
	}
	if config.OutboxEnabled {
		outbox := application.NewOutboxDispatcher(application.OutboxDispatcherDeps{
			Repository: outboxDispatch,
			Publisher:  integrationEvents,
			Clock:      clock,
		}, application.OutboxDispatcherConfig{
			DispatcherID:    "zhicore-content:outbox-dispatcher",
			BatchSize:       50,
			StaleClaimAfter: 5 * time.Minute,
			RetryBackoff:    time.Minute,
			DeadThreshold:   5,
		})
		workers[3] = enabledWorkerDescriptor("content-outbox-dispatcher", pollingWorker{name: "content-outbox-dispatcher", interval: interval, runUntilIdle: outbox.RunUntilIdle})
	}
	return workers
}

func disabledWorkerDescriptor(name, reason string) WorkerDescriptor {
	return WorkerDescriptor{Name: name, Enabled: false, DisabledReason: reason}
}

func enabledWorkerDescriptor(name string, runner Worker) WorkerDescriptor {
	return WorkerDescriptor{Name: name, Enabled: true, Checker: healthyWorkerChecker{}, Runner: runner}
}

type pollingWorker struct {
	name         string
	interval     time.Duration
	runUntilIdle func(context.Context) error
}

func (w pollingWorker) Run(ctx context.Context) error {
	if w.interval <= 0 {
		w.interval = time.Minute
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := w.runUntilIdle(ctx); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			return fmt.Errorf("run %s: %w", w.name, err)
		}
		timer := time.NewTimer(w.interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

type healthyWorkerChecker struct{}

func (healthyWorkerChecker) Check(context.Context) error {
	return nil
}

func healthHandler(details HealthDetails) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Health exposes dependency wiring and worker availability without
		// pretending disabled cleanup, repair or outbox workers are running.
		c.JSON(http.StatusOK, details)
	}
}

func readyHandler(details HealthDetails, checks HealthCheckers, workers []WorkerDescriptor) gin.HandlerFunc {
	return func(c *gin.Context) {
		failures := readinessFailures(c.Request.Context(), checks, workers)
		if len(failures) > 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"service":  details.Service,
				"status":   "not_ready",
				"failures": failures,
				"workers":  workers,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"service": details.Service,
			"status":  "ready",
			"details": details,
		})
	}
}

func readinessFailures(ctx context.Context, checks HealthCheckers, workers []WorkerDescriptor) []string {
	failures := make([]string, 0, 4)
	if checks.Lifecycle != nil && checkFailed(ctx, checks.Lifecycle) {
		failures = append(failures, "lifecycle not ready")
	}
	if checkFailed(ctx, checks.Postgres) {
		failures = append(failures, "postgres unavailable")
	}
	if checkFailed(ctx, checks.Mongo) {
		failures = append(failures, "mongo unavailable")
	}
	if checkFailed(ctx, checks.RabbitMQ) {
		failures = append(failures, "rabbitmq unavailable")
	}
	for _, worker := range workers {
		if !worker.Enabled {
			continue
		}
		if strings.TrimSpace(worker.Name) == "" {
			failures = append(failures, "worker unavailable")
			continue
		}
		if worker.Checker == nil || checkFailed(ctx, worker.Checker) {
			failures = append(failures, worker.Name+" unavailable")
		}
	}
	return failures
}

func checkFailed(ctx context.Context, checker HealthChecker) bool {
	if checker == nil {
		return true
	}
	return checker.Check(ctx) != nil
}
