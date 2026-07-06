package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthLiveDoesNotProbeDependencies(t *testing.T) {
	probe := &fakeProbe{name: "postgres", err: context.Canceled}
	handler := NewHealthHandler(HealthDeps{Dependencies: []DependencyCheck{probe}})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("live status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if probe.calls != 0 {
		t.Fatalf("live probe calls = %d, want 0", probe.calls)
	}
}

func TestHealthReadyChecksDependenciesAndEnabledWorkers(t *testing.T) {
	postgres := &fakeProbe{name: "postgres"}
	redis := &fakeProbe{name: "redis"}
	rabbit := &fakeProbe{name: "rabbitmq"}
	worker := WorkerDescriptor{Name: "cleanup_consumed_events", Enabled: true, Ready: true}
	handler := NewHealthHandler(HealthDeps{Dependencies: []DependencyCheck{postgres, redis, rabbit}, Workers: []WorkerDescriptor{worker}})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ready status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if postgres.calls != 1 || redis.calls != 1 || rabbit.calls != 1 {
		t.Fatalf("probe calls postgres=%d redis=%d rabbit=%d", postgres.calls, redis.calls, rabbit.calls)
	}
}

func TestHealthReadyFailsWhenEnabledWorkerNotReady(t *testing.T) {
	handler := NewHealthHandler(HealthDeps{Workers: []WorkerDescriptor{{Name: "campaign_shard", Enabled: true, Ready: false}}})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status = %d, want 503; body=%s", rr.Code, rr.Body.String())
	}
}

type fakeProbe struct {
	name  string
	err   error
	calls int
}

func (f *fakeProbe) Name() string {
	return f.name
}

func (f *fakeProbe) Check(ctx context.Context) error {
	f.calls++
	return f.err
}
