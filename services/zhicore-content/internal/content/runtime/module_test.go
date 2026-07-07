package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
)

func TestBuildRejectsMissingRuntimeDependencies(t *testing.T) {
	valid := validDeps(t)

	testCases := []struct {
		name   string
		mutate func(*Deps)
		want   string
	}{
		{name: "config", mutate: func(deps *Deps) { deps.Config = nil }, want: "Config"},
		{name: "postgres", mutate: func(deps *Deps) { deps.PostgresDB = nil }, want: "PostgresDB"},
		{name: "mongo", mutate: func(deps *Deps) { deps.BodyCollection = nil }, want: "BodyCollection"},
		{name: "redis health", mutate: func(deps *Deps) { deps.Health.Redis = nil }, want: "Redis health"},
		{name: "parser", mutate: func(deps *Deps) { deps.Parser = nil }, want: "Parser"},
		{name: "outbox", mutate: func(deps *Deps) { deps.Outbox = nil }, want: "Outbox"},
		{name: "rate limiter", mutate: func(deps *Deps) { deps.RateLimiter = nil }, want: "RateLimiter"},
		{name: "observer", mutate: func(deps *Deps) { deps.Observer = nil }, want: "Observer"},
		{name: "clock", mutate: func(deps *Deps) { deps.Clock = nil }, want: "Clock"},
		{name: "users", mutate: func(deps *Deps) { deps.Users = nil }, want: "Users"},
		{name: "files", mutate: func(deps *Deps) { deps.Files = nil }, want: "Files"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			deps := valid
			tc.mutate(&deps)
			_, err := Build(deps)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Build() error = %v, want mention %s", err, tc.want)
			}
		})
	}
}

func TestBuildReturnsHTTPHandlerWorkerDescriptionsAndHealthDetails(t *testing.T) {
	module, err := Build(validDeps(t))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if module.HTTPHandler == nil {
		t.Fatal("HTTPHandler = nil")
	}
	if module.HealthDetails.Service != "zhicore-content" || module.HealthDetails.Postgres != "configured" ||
		module.HealthDetails.Mongo != "configured" || module.HealthDetails.Redis != "configured" || module.HealthDetails.BodyParser != "v1" {
		t.Fatalf("health details = %#v", module.HealthDetails)
	}
	if len(module.Workers) != 3 {
		t.Fatalf("workers = %d, want cleanup/repair/outbox descriptors", len(module.Workers))
	}
	for _, worker := range module.Workers {
		if worker.Enabled || strings.TrimSpace(worker.DisabledReason) == "" {
			t.Fatalf("worker descriptor = %#v, want disabled with reason", worker)
		}
	}

	for _, path := range []string{"/health/live", "/health/ready"} {
		rec := httptest.NewRecorder()
		module.HTTPHandler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "zhicore-content") {
			t.Fatalf("%s body = %s, want service details", path, rec.Body.String())
		}
	}
}

func TestBuildWiresAdminOutboxRepository(t *testing.T) {
	module, err := Build(validDeps(t))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/content/outbox-events?status=failed", nil)
	req.Header.Set("X-User-Id", "1001")
	req.Header.Set("X-User-Roles", "admin")
	module.HTTPHandler.ServeHTTP(rec, req)

	if rec.Code == http.StatusServiceUnavailable {
		t.Fatalf("admin outbox route returned service unavailable, want runtime Admin repository wired; body=%s", rec.Body.String())
	}
}

func TestBuildWiresObserverIntoRateLimitedService(t *testing.T) {
	limiter := &recordingRateLimiter{
		decision: ports.RateLimitDecision{
			Outcome:   ports.RateLimitOutcomeAllow,
			LimitType: ports.RateLimitTypePublicRead,
			Reason:    "allow",
			Fallback:  ports.RateLimitFallbackNone,
		},
	}
	observer := &recordingContentObserver{}
	deps := validDeps(t)
	deps.RateLimiter = limiter
	deps.Observer = observer

	module, err := Build(deps)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	rec := httptest.NewRecorder()
	module.HTTPHandler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/posts?limit=1", nil))

	if limiter.calls != 1 {
		t.Fatalf("rate limiter calls = %d, want 1", limiter.calls)
	}
	if limiter.lastRequest.LimitType != ports.RateLimitTypePublicRead {
		t.Fatalf("rate limiter request = %#v, want public_read", limiter.lastRequest)
	}
	if len(observer.decisions) != 1 || observer.decisions[0].LimitType != ports.RateLimitTypePublicRead {
		t.Fatalf("observer decisions = %#v, want public_read decision", observer.decisions)
	}
}

func validDeps(t *testing.T) Deps {
	t.Helper()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := stubOutbox{}
	return Deps{
		Config:         &Config{ServiceName: "zhicore-content"},
		PostgresDB:     db,
		BodyCollection: &drivermongo.Collection{},
		Health: HealthCheckers{
			Postgres: healthyCheck("postgres"),
			Mongo:    healthyCheck("mongo"),
			Redis:    healthyCheck("redis"),
			RabbitMQ: healthyCheck("rabbitmq"),
		},
		Parser:            stubBodyParser{},
		Outbox:            store,
		IntegrationEvents: stubIntegrationEvents{},
		RateLimiter:       &recordingRateLimiter{decision: ports.RateLimitDecision{Outcome: ports.RateLimitOutcomeAllow, Reason: "allow"}},
		Observer:          &recordingContentObserver{},
		Clock:             fixedClock{now: time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)},
		Users:             stubUsers{},
		Files:             stubFiles{},
	}
}

type stubBodyParser struct{}

func (stubBodyParser) Parse(context.Context, ports.PostBodyWriteInput) (ports.NormalizedBody, error) {
	return ports.NormalizedBody{}, nil
}

type stubOutbox struct{}

func (stubOutbox) Append(context.Context, ports.Tx, ports.OutboxEvent) error { return nil }

type stubIntegrationEvents struct{}

func (stubIntegrationEvents) PublishIntegrationEvent(context.Context, ports.OutboxEvent) error {
	return nil
}

type recordingRateLimiter struct {
	decision    ports.RateLimitDecision
	calls       int
	lastRequest ports.RateLimitRequest
}

func (r *recordingRateLimiter) Check(ctx context.Context, request ports.RateLimitRequest) ports.RateLimitDecision {
	r.calls++
	r.lastRequest = request
	decision := r.decision
	if decision.LimitType == "" {
		decision.LimitType = request.LimitType
	}
	return decision
}

type recordingContentObserver struct {
	decisions []ports.RateLimitDecision
}

func (o *recordingContentObserver) ObserveRateLimitDecision(ctx context.Context, decision ports.RateLimitDecision) {
	o.decisions = append(o.decisions, decision)
}

type stubUsers struct{}

func (stubUsers) GetOwnerSnapshot(context.Context, int64) (ports.OwnerSnapshot, error) {
	return ports.OwnerSnapshot{DisplayName: "author", ProfileVersion: 1}, nil
}

type stubFiles struct{}

func (stubFiles) ValidateBodyMediaRefs(context.Context, []ports.MediaRef) error { return nil }

func (stubFiles) ValidateCoverFile(context.Context, string) error { return nil }

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time { return c.now }

var _ ports.Clock = fixedClock{}
var _ ports.BodyParserRegistry = stubBodyParser{}
var _ ports.OutboxPublisher = stubOutbox{}
var _ ports.IntegrationEventPublisher = stubIntegrationEvents{}
var _ ports.RateLimiter = (*recordingRateLimiter)(nil)
var _ ports.ContentObserver = (*recordingContentObserver)(nil)
var _ ports.UserProfileClient = stubUsers{}
var _ ports.FileResourceClient = stubFiles{}
