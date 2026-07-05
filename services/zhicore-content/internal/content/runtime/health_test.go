package runtime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthReadinessChecksRequiredDependencies(t *testing.T) {
	postgres := &recordingHealthChecker{name: "postgres"}
	mongo := &recordingHealthChecker{name: "mongo"}
	rabbitmq := &recordingHealthChecker{name: "rabbitmq"}
	worker := &recordingHealthChecker{name: "worker"}
	deps := validDeps(t)
	deps.Health = HealthCheckers{Postgres: postgres, Mongo: mongo, RabbitMQ: rabbitmq}
	deps.Workers = []WorkerDescriptor{{Name: "content-outbox-dispatcher", Enabled: true, Checker: worker}}

	module, err := Build(deps)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	live := httptest.NewRecorder()
	module.HTTPHandler.ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if live.Code != http.StatusOK {
		t.Fatalf("/health/live status = %d, want 200", live.Code)
	}
	if postgres.calls != 0 || mongo.calls != 0 || rabbitmq.calls != 0 || worker.calls != 0 {
		t.Fatalf("live health called dependencies: postgres=%d mongo=%d rabbitmq=%d worker=%d", postgres.calls, mongo.calls, rabbitmq.calls, worker.calls)
	}

	ready := httptest.NewRecorder()
	module.HTTPHandler.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if ready.Code != http.StatusOK {
		t.Fatalf("/health/ready status = %d body = %s, want 200", ready.Code, ready.Body.String())
	}
	if postgres.calls != 1 || mongo.calls != 1 || rabbitmq.calls != 1 || worker.calls != 1 {
		t.Fatalf("ready health calls = postgres=%d mongo=%d rabbitmq=%d worker=%d, want all once", postgres.calls, mongo.calls, rabbitmq.calls, worker.calls)
	}
}

func TestHealthReadinessReturnsUnavailableForDependencyFailure(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(*Deps)
		want   string
	}{
		{
			name: "lifecycle",
			mutate: func(deps *Deps) {
				deps.Health.Lifecycle = failingCheck("lifecycle", "server is shutting down")
			},
			want: "lifecycle",
		},
		{
			name: "postgres",
			mutate: func(deps *Deps) {
				deps.Health.Postgres = failingCheck("postgres", "postgres://content:secret@db.internal:5432/content")
			},
			want: "postgres",
		},
		{
			name: "mongo",
			mutate: func(deps *Deps) {
				deps.Health.Mongo = failingCheck("mongo", "mongodb://content:secret@mongo.internal:27017")
			},
			want: "mongo",
		},
		{
			name: "rabbitmq",
			mutate: func(deps *Deps) {
				deps.Health.RabbitMQ = failingCheck("rabbitmq", "amqp://content:secret@mq.internal:5672/")
			},
			want: "rabbitmq",
		},
		{
			name: "enabled worker",
			mutate: func(deps *Deps) {
				deps.Workers = []WorkerDescriptor{{Name: "content-body-cleanup", Enabled: true, Checker: failingCheck("worker", "postgres://content:secret@db.internal:5432/content")}}
			},
			want: "content-body-cleanup",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			deps := validDeps(t)
			tc.mutate(&deps)

			module, err := Build(deps)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}

			live := httptest.NewRecorder()
			module.HTTPHandler.ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/health/live", nil))
			if live.Code != http.StatusOK {
				t.Fatalf("/health/live status = %d, want 200 even when dependency fails", live.Code)
			}

			ready := httptest.NewRecorder()
			module.HTTPHandler.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
			if ready.Code != http.StatusServiceUnavailable {
				t.Fatalf("/health/ready status = %d body = %s, want 503", ready.Code, ready.Body.String())
			}
			if !strings.Contains(ready.Body.String(), tc.want) {
				t.Fatalf("/health/ready body = %s, want mention %s", ready.Body.String(), tc.want)
			}
			if strings.Contains(ready.Body.String(), "secret") || strings.Contains(ready.Body.String(), "content:secret@") {
				t.Fatalf("/health/ready leaked sensitive dependency error: %s", ready.Body.String())
			}
		})
	}
}

type recordingHealthChecker struct {
	name  string
	err   error
	calls int
}

func (c *recordingHealthChecker) Check(context.Context) error {
	c.calls++
	return c.err
}

func healthyCheck(name string) HealthChecker {
	return &recordingHealthChecker{name: name}
}

func failingCheck(name, message string) HealthChecker {
	return &recordingHealthChecker{name: name, err: errors.New(message)}
}
