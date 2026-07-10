package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/application"
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

func TestModuleStartStopTracksWorkerReadiness(t *testing.T) {
	worker := &blockingWorker{name: "campaign_shard", started: make(chan struct{}), stopped: make(chan struct{})}
	module, err := Build(Deps{
		Service: fakeNotificationService{},
		Workers: []WorkerRunner{worker},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	<-worker.started
	assertWorkerReady(t, module, "campaign_shard", true)

	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	<-worker.stopped
	assertWorkerReady(t, module, "campaign_shard", false)
}

func TestModuleStartStopTracksMultipleWorkerReadiness(t *testing.T) {
	workers := []*blockingWorker{
		{name: "campaign_shard_1", started: make(chan struct{}), stopped: make(chan struct{})},
		{name: "campaign_shard_2", started: make(chan struct{}), stopped: make(chan struct{})},
		{name: "campaign_shard_3", started: make(chan struct{}), stopped: make(chan struct{})},
	}
	module, err := Build(Deps{
		Service: fakeNotificationService{},
		Workers: []WorkerRunner{
			workers[0],
			workers[1],
			workers[2],
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	for _, worker := range workers {
		waitForWorkerSignal(t, worker.started)
		assertWorkerReady(t, module, worker.name, true)
	}

	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	for _, worker := range workers {
		waitForWorkerSignal(t, worker.stopped)
		assertWorkerReady(t, module, worker.name, false)
	}
}

func TestModuleRecoversPanickingWorkerAndMarksNotReady(t *testing.T) {
	worker := &panicWorker{name: "campaign_shard", done: make(chan struct{})}
	module, err := Build(Deps{
		Service: fakeNotificationService{},
		Workers: []WorkerRunner{worker},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	<-worker.done
	assertWorkerReady(t, module, "campaign_shard", false)
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestLoopWorkerContinuesAfterRecoverableRunOnceError(t *testing.T) {
	calls := make(chan int, 2)
	worker := NewLoopWorker("campaign_shard", time.Millisecond, func(ctx context.Context) error {
		call := len(calls) + 1
		calls <- call
		if call == 1 {
			return context.DeadlineExceeded
		}
		return ctx.Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-calls
		<-calls
		cancel()
	}()

	err := worker.Run(ctx)
	if err != context.Canceled {
		t.Fatalf("Run() error = %v, want context.Canceled after retry loop", err)
	}
}

func assertWorkerReady(t *testing.T, module *Module, name string, ready bool) {
	t.Helper()
	for _, worker := range module.WorkerDescriptors() {
		if worker.Name == name {
			if worker.Ready != ready || !worker.Enabled {
				t.Fatalf("worker descriptor = %+v, want enabled ready=%v", worker, ready)
			}
			return
		}
	}
	t.Fatalf("worker %s not found in descriptors %+v", name, module.WorkerDescriptors())
}

func waitForWorkerSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for worker signal")
	}
}

type fakeProbe struct {
	name  string
	err   error
	calls int
}

type fakeNotificationService struct{}

func (fakeNotificationService) MarkNotificationRead(context.Context, application.MarkNotificationReadCommand) (application.MarkNotificationReadResult, error) {
	return application.MarkNotificationReadResult{}, nil
}
func (fakeNotificationService) MarkAllNotificationsRead(context.Context, application.MarkAllNotificationsReadCommand) (application.MarkAllNotificationsReadResult, error) {
	return application.MarkAllNotificationsReadResult{}, nil
}
func (fakeNotificationService) GetUnreadCount(context.Context, application.GetUnreadCountQuery) (application.UnreadCountResult, error) {
	return application.UnreadCountResult{}, nil
}
func (fakeNotificationService) GetUnreadBreakdown(context.Context, application.GetUnreadBreakdownQuery) (application.UnreadBreakdownResult, error) {
	return application.UnreadBreakdownResult{}, nil
}
func (fakeNotificationService) ListAggregatedNotifications(context.Context, application.ListNotificationsQuery) (application.NotificationPage, error) {
	return application.NotificationPage{}, nil
}
func (fakeNotificationService) ListNotificationGroupActors(context.Context, application.ListNotificationGroupActorsQuery) (application.NotificationActorPage, error) {
	return application.NotificationActorPage{}, nil
}
func (fakeNotificationService) MarkNotificationGroupRead(context.Context, application.MarkNotificationGroupReadCommand) (application.MarkNotificationGroupReadResult, error) {
	return application.MarkNotificationGroupReadResult{}, nil
}
func (fakeNotificationService) GetNotificationPreferences(context.Context, application.GetNotificationPreferencesQuery) (application.NotificationPreferencesResult, error) {
	return application.NotificationPreferencesResult{}, nil
}
func (fakeNotificationService) UpdateNotificationPreferences(context.Context, application.UpdateNotificationPreferencesCommand) (application.NotificationPreferencesResult, error) {
	return application.NotificationPreferencesResult{}, nil
}
func (fakeNotificationService) GetNotificationDND(context.Context, application.GetNotificationDNDQuery) (application.NotificationDNDResult, error) {
	return application.NotificationDNDResult{}, nil
}
func (fakeNotificationService) UpdateNotificationDND(context.Context, application.UpdateNotificationDNDCommand) (application.NotificationDNDResult, error) {
	return application.NotificationDNDResult{}, nil
}
func (fakeNotificationService) GetAuthorSubscription(context.Context, application.GetAuthorSubscriptionQuery) (application.AuthorSubscriptionResult, error) {
	return application.AuthorSubscriptionResult{}, nil
}
func (fakeNotificationService) UpdateAuthorSubscription(context.Context, application.UpdateAuthorSubscriptionCommand) (application.AuthorSubscriptionResult, error) {
	return application.AuthorSubscriptionResult{}, nil
}
func (fakeNotificationService) ListDeliveries(context.Context, application.ListDeliveriesQuery) (application.DeliveryPage, error) {
	return application.DeliveryPage{}, nil
}
func (fakeNotificationService) RetryDelivery(context.Context, application.RetryDeliveryCommand) (application.DeliveryRetryResult, error) {
	return application.DeliveryRetryResult{}, nil
}

type blockingWorker struct {
	name    string
	started chan struct{}
	stopped chan struct{}
}

func (w *blockingWorker) Name() string {
	return w.name
}

func (w *blockingWorker) Run(ctx context.Context) error {
	close(w.started)
	<-ctx.Done()
	close(w.stopped)
	return ctx.Err()
}

type panicWorker struct {
	name string
	done chan struct{}
}

func (w *panicWorker) Name() string {
	return w.name
}

func (w *panicWorker) Run(context.Context) error {
	defer close(w.done)
	panic("campaign shard panic")
}

func (f *fakeProbe) Name() string {
	return f.name
}

func (f *fakeProbe) Check(ctx context.Context) error {
	f.calls++
	return f.err
}
