package runtime

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

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
	Config         *Config
	PostgresDB     *sql.DB
	BodyCollection *drivermongo.Collection
	Health         HealthCheckers
	Workers        []WorkerDescriptor
	Parser         ports.BodyParserRegistry
	Outbox         ports.OutboxPublisher
	Clock          ports.Clock
	Users          ports.UserProfileClient
	Files          ports.FileResourceClient
}

type Module struct {
	HTTPHandler   *gin.Engine
	Workers       []WorkerDescriptor
	HealthDetails HealthDetails
}

type WorkerDescriptor struct {
	Name           string        `json:"name"`
	Enabled        bool          `json:"enabled"`
	DisabledReason string        `json:"disabledReason,omitempty"`
	Checker        HealthChecker `json:"-"`
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
	service := application.NewService(application.Deps{
		Posts:   store,
		Queries: store,
		Bodies:  bodyStore,
		Cleanup: cleanupStore,
		Repair:  repairStore,
		Outbox:  deps.Outbox,
		Users:   deps.Users,
		Files:   deps.Files,
		Tx:      contentpostgres.NewTransactionRunner(deps.PostgresDB),
		Parser:  deps.Parser,
		Clock:   deps.Clock,
	})

	workers := configuredWorkerDescriptors(deps.Workers)
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

func configuredWorkerDescriptors(workers []WorkerDescriptor) []WorkerDescriptor {
	if len(workers) == 0 {
		return disabledWorkerDescriptors()
	}
	copied := make([]WorkerDescriptor, len(workers))
	copy(copied, workers)
	return copied
}

func disabledWorkerDescriptors() []WorkerDescriptor {
	reason := "disabled until dedicated worker runtime is implemented"
	return []WorkerDescriptor{
		{Name: "content-body-cleanup", Enabled: false, DisabledReason: reason},
		{Name: "content-body-repair", Enabled: false, DisabledReason: reason},
		{Name: "content-outbox-dispatcher", Enabled: false, DisabledReason: reason},
	}
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
