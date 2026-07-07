package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	notificationhttp "github.com/architectcgz/zhicore-go/services/zhicore-notification/api/http"
	"github.com/gin-gonic/gin"
)

type DependencyCheck interface {
	Name() string
	Check(context.Context) error
}

type HealthDeps struct {
	ServiceName          string
	Dependencies         []DependencyCheck
	Workers              []WorkerDescriptor
	WorkerStatusProvider func() []WorkerDescriptor
}

type WorkerDescriptor struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Ready   bool   `json:"ready"`
}

type Deps struct {
	Service notificationhttp.Service
	Health  HealthDeps
	Workers []WorkerRunner
}

type Module struct {
	HTTPHandler http.Handler
	Health      HealthDeps
	workers     *workerSupervisor
}

func Build(deps Deps) (*Module, error) {
	if deps.Service == nil {
		return nil, fmt.Errorf("notification runtime service dependency is required")
	}
	supervisor := newWorkerSupervisor(deps.Workers)
	router := notificationhttp.NewHandler(deps.Service)
	health := normalizeHealthDeps(deps.Health)
	if supervisor != nil {
		health.WorkerStatusProvider = supervisor.Descriptors
	}
	router.GET("/health/live", gin.WrapH(NewHealthHandler(health)))
	router.GET("/health/ready", gin.WrapH(NewHealthHandler(health)))
	return &Module{HTTPHandler: router, Health: health, workers: supervisor}, nil
}

func (m *Module) Start(ctx context.Context) error {
	if m.workers == nil {
		return nil
	}
	return m.workers.Start(ctx)
}

func (m *Module) Stop(ctx context.Context) error {
	if m.workers == nil {
		return nil
	}
	return m.workers.Stop(ctx)
}

func (m *Module) WorkerDescriptors() []WorkerDescriptor {
	if m.workers == nil {
		return currentWorkers(m.Health)
	}
	return m.workers.Descriptors()
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
					"workers":  currentWorkers(deps),
				})
				return
			}
			writeHealthJSON(w, http.StatusOK, map[string]any{
				"service": deps.ServiceName,
				"status":  "ready",
				"workers": currentWorkers(deps),
			})
		default:
			http.NotFound(w, r)
		}
	})
}

func currentWorkers(deps HealthDeps) []WorkerDescriptor {
	if deps.WorkerStatusProvider != nil {
		return deps.WorkerStatusProvider()
	}
	return deps.Workers
}

func normalizeHealthDeps(deps HealthDeps) HealthDeps {
	if strings.TrimSpace(deps.ServiceName) == "" {
		deps.ServiceName = "zhicore-notification"
	}
	return deps
}

func readinessFailures(ctx context.Context, deps HealthDeps) []string {
	workers := currentWorkers(deps)
	failures := make([]string, 0, len(deps.Dependencies)+len(workers))
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
	for _, worker := range workers {
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

type WorkerRunner interface {
	Name() string
	Run(context.Context) error
}

type LoopWorker struct {
	name     string
	interval time.Duration
	runOnce  func(context.Context) error
}

func NewLoopWorker(name string, interval time.Duration, runOnce func(context.Context) error) *LoopWorker {
	if interval <= 0 {
		interval = time.Second
	}
	return &LoopWorker{name: strings.TrimSpace(name), interval: interval, runOnce: runOnce}
}

func (w *LoopWorker) Name() string {
	return w.name
}

func (w *LoopWorker) Run(ctx context.Context) error {
	if w.runOnce == nil {
		return fmt.Errorf("worker %s run function is required", w.name)
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := w.runOnceSafely(ctx); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
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

func (w *LoopWorker) runOnceSafely(ctx context.Context) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("worker %s panic: %v", w.name, recovered)
		}
	}()
	return w.runOnce(ctx)
}

type workerSupervisor struct {
	workers []WorkerRunner
	mu      sync.Mutex
	status  map[string]WorkerDescriptor
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

func newWorkerSupervisor(workers []WorkerRunner) *workerSupervisor {
	if len(workers) == 0 {
		return nil
	}
	status := make(map[string]WorkerDescriptor, len(workers))
	for _, worker := range workers {
		if worker == nil {
			continue
		}
		name := strings.TrimSpace(worker.Name())
		if name == "" {
			name = "worker"
		}
		status[name] = WorkerDescriptor{Name: name, Enabled: true, Ready: false}
	}
	return &workerSupervisor{workers: workers, status: status}
}

func (s *workerSupervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.started = true
	s.mu.Unlock()

	for _, worker := range s.workers {
		if worker == nil {
			continue
		}
		name := strings.TrimSpace(worker.Name())
		if name == "" {
			name = "worker"
		}
		s.wg.Add(1)
		go s.runWorker(workerCtx, name, worker)
	}
	return nil
}

func (s *workerSupervisor) runWorker(ctx context.Context, name string, worker WorkerRunner) {
	defer s.wg.Done()
	s.setReady(name, true)
	defer s.setReady(name, false)
	defer func() {
		// Worker panic is contained at the owner boundary so readiness flips to
		// not ready and the process can still drain HTTP shutdown cleanly.
		_ = recover()
	}()
	_ = worker.Run(ctx)
}

func (s *workerSupervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *workerSupervisor) Descriptors() []WorkerDescriptor {
	s.mu.Lock()
	defer s.mu.Unlock()
	descriptors := make([]WorkerDescriptor, 0, len(s.status))
	for _, descriptor := range s.status {
		descriptors = append(descriptors, descriptor)
	}
	return descriptors
}

func (s *workerSupervisor) setReady(name string, ready bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	descriptor := s.status[name]
	descriptor.Name = name
	descriptor.Enabled = true
	descriptor.Ready = ready
	s.status[name] = descriptor
}

func writeHealthJSON(w http.ResponseWriter, status int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
