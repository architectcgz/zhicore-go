package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	notificationhttp "github.com/architectcgz/zhicore-go/services/zhicore-notification/api/http"
	"github.com/gin-gonic/gin"
)

type DependencyCheck interface {
	Name() string
	Check(context.Context) error
}

type HealthDeps struct {
	ServiceName  string
	Dependencies []DependencyCheck
	Workers      []WorkerDescriptor
}

type WorkerDescriptor struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Ready   bool   `json:"ready"`
}

type Deps struct {
	Service notificationhttp.Service
	Health  HealthDeps
}

type Module struct {
	HTTPHandler http.Handler
	Health      HealthDeps
}

func Build(deps Deps) (*Module, error) {
	if deps.Service == nil {
		return nil, fmt.Errorf("notification runtime service dependency is required")
	}
	router := notificationhttp.NewHandler(deps.Service)
	health := normalizeHealthDeps(deps.Health)
	router.GET("/health/live", gin.WrapH(NewHealthHandler(health)))
	router.GET("/health/ready", gin.WrapH(NewHealthHandler(health)))
	return &Module{HTTPHandler: router, Health: health}, nil
}

func NewHealthHandler(deps HealthDeps) http.Handler {
	deps = normalizeHealthDeps(deps)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health/live":
			writeHealthJSON(w, http.StatusOK, map[string]any{
				"service": deps.ServiceName,
				"status":  "live",
			})
		case "/health/ready":
			failures := readinessFailures(r.Context(), deps)
			if len(failures) > 0 {
				writeHealthJSON(w, http.StatusServiceUnavailable, map[string]any{
					"service":  deps.ServiceName,
					"status":   "not_ready",
					"failures": failures,
					"workers":  deps.Workers,
				})
				return
			}
			writeHealthJSON(w, http.StatusOK, map[string]any{
				"service": deps.ServiceName,
				"status":  "ready",
				"workers": deps.Workers,
			})
		default:
			http.NotFound(w, r)
		}
	})
}

func normalizeHealthDeps(deps HealthDeps) HealthDeps {
	if strings.TrimSpace(deps.ServiceName) == "" {
		deps.ServiceName = "zhicore-notification"
	}
	return deps
}

func readinessFailures(ctx context.Context, deps HealthDeps) []string {
	failures := make([]string, 0, len(deps.Dependencies)+len(deps.Workers))
	for _, dependency := range deps.Dependencies {
		if dependency == nil {
			failures = append(failures, "dependency unavailable")
			continue
		}
		name := strings.TrimSpace(dependency.Name())
		if name == "" {
			name = "dependency"
		}
		// Ready checks intentionally expose only stable dependency names. Raw
		// driver errors can contain DSNs, credentials, URLs or broker details.
		if err := dependency.Check(ctx); err != nil {
			failures = append(failures, name+" unavailable")
		}
	}
	for _, worker := range deps.Workers {
		if !worker.Enabled {
			continue
		}
		name := strings.TrimSpace(worker.Name)
		if name == "" {
			name = "worker"
		}
		if !worker.Ready {
			failures = append(failures, name+" unavailable")
		}
	}
	return failures
}

func writeHealthJSON(w http.ResponseWriter, status int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
