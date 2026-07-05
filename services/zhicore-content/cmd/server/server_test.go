package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"reflect"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestContentServerLifecycleAppliesHTTPTimeouts(t *testing.T) {
	cfg := DefaultContentServerConfig()
	cfg.HTTP.Addr = "127.0.0.1:0"
	cfg.HTTP.ReadHeaderTimeout = 3 * time.Second
	cfg.HTTP.ReadTimeout = 4 * time.Second
	cfg.HTTP.WriteTimeout = 5 * time.Second
	cfg.HTTP.IdleTimeout = 6 * time.Second

	server := newContentHTTPServer(cfg, http.NewServeMux())

	if server.Addr != cfg.HTTP.Addr ||
		server.ReadHeaderTimeout != cfg.HTTP.ReadHeaderTimeout ||
		server.ReadTimeout != cfg.HTTP.ReadTimeout ||
		server.WriteTimeout != cfg.HTTP.WriteTimeout ||
		server.IdleTimeout != cfg.HTTP.IdleTimeout {
		t.Fatalf("http.Server = %#v, want configured addr and timeouts", server)
	}
}

func TestContentServerLifecycleHandlesTerminationSignal(t *testing.T) {
	var orderMu sync.Mutex
	order := make([]string, 0, 5)
	record := func(event string) {
		orderMu.Lock()
		defer orderMu.Unlock()
		order = append(order, event)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	cfg := DefaultContentServerConfig()
	cfg.HTTP.Addr = listener.Addr().String()
	cfg.HTTP.ShutdownTimeout = time.Second
	signals := make(chan os.Signal, 1)
	runtime := ContentServerRuntime{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		Readiness: &recordingReadiness{record: record, ready: true},
		Workers:   &recordingWorkers{record: record},
		Closers: []namedCloser{
			{name: "postgres", closer: recordingCloser{record: record, name: "postgres"}},
			{name: "mongo", closer: recordingCloser{record: record, name: "mongo"}},
		},
	}

	done := make(chan error, 1)
	go func() {
		done <- runContentServer(context.Background(), cfg, listener, runtime, signals)
	}()

	waitForHTTPReady(t, listener.Addr().String())
	signals <- syscall.SIGTERM

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runContentServer() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runContentServer() did not stop after SIGTERM")
	}

	wantOrder := []string{"readiness:ready", "workers:start", "readiness:not-ready", "workers:stop", "workers:wait", "close:postgres", "close:mongo"}
	orderMu.Lock()
	gotOrder := append([]string(nil), order...)
	orderMu.Unlock()
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("shutdown order = %v, want %v", gotOrder, wantOrder)
	}
}

func TestContentServerLifecycleUsesShutdownTimeoutAfterParentContextCancel(t *testing.T) {
	var waited bool
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	cfg := DefaultContentServerConfig()
	cfg.HTTP.Addr = listener.Addr().String()
	cfg.HTTP.ShutdownTimeout = time.Second
	ctx, cancel := context.WithCancel(context.Background())
	runtime := ContentServerRuntime{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		Readiness: &recordingReadiness{ready: true},
		Workers: workerLifecycleFunc{
			start: func(context.Context) error { return nil },
			stop:  func() {},
			wait: func(ctx context.Context) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				waited = true
				return nil
			},
		},
	}

	done := make(chan error, 1)
	go func() {
		done <- runContentServer(ctx, cfg, listener, runtime, make(chan os.Signal))
	}()

	waitForHTTPReady(t, listener.Addr().String())
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runContentServer() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runContentServer() did not stop after parent context cancel")
	}
	if !waited {
		t.Fatal("worker Wait was not called with an active shutdown timeout context")
	}
}

type recordingReadiness struct {
	record func(string)
	ready  bool
}

func (r *recordingReadiness) MarkReady() {
	r.ready = true
	if r.record != nil {
		r.record("readiness:ready")
	}
}

func (r *recordingReadiness) MarkNotReady() {
	r.ready = false
	if r.record != nil {
		r.record("readiness:not-ready")
	}
}

func (r *recordingReadiness) IsReady() bool {
	return r.ready
}

type recordingWorkers struct {
	record func(string)
}

func (w *recordingWorkers) StopAcceptingNewWork() {
	w.record("workers:stop")
}

func (w *recordingWorkers) Start(context.Context) error {
	w.record("workers:start")
	return nil
}

func (w *recordingWorkers) Wait(context.Context) error {
	w.record("workers:wait")
	return nil
}

type workerLifecycleFunc struct {
	start func(context.Context) error
	stop  func()
	wait  func(context.Context) error
}

func (w workerLifecycleFunc) Start(ctx context.Context) error {
	if w.start == nil {
		return nil
	}
	return w.start(ctx)
}

func (w workerLifecycleFunc) StopAcceptingNewWork() {
	if w.stop != nil {
		w.stop()
	}
}

func (w workerLifecycleFunc) Wait(ctx context.Context) error {
	if w.wait == nil {
		return nil
	}
	return w.wait(ctx)
}

type recordingCloser struct {
	record func(string)
	name   string
	err    error
}

func (c recordingCloser) Close() error {
	c.record("close:" + c.name)
	return c.err
}

func waitForHTTPReady(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			time.Sleep(10 * time.Millisecond)
		}
	}
	t.Fatalf("server at %s did not start listening", addr)
}
