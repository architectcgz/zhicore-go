package runtime

import (
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

type Deps struct {
	Config         *Config
	PostgresDB     *sql.DB
	BodyCollection *drivermongo.Collection
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
	Name           string `json:"name"`
	Enabled        bool   `json:"enabled"`
	DisabledReason string `json:"disabledReason,omitempty"`
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

	workers := disabledWorkerDescriptors()
	health := HealthDetails{
		Service:    serviceName(deps.Config),
		Postgres:   "configured",
		Mongo:      "configured",
		BodyParser: "v1",
		Workers:    workers,
	}
	root := contenthttp.NewHandler(service)
	root.GET("/health/live", healthHandler(health))
	root.GET("/health/ready", healthHandler(health))

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
