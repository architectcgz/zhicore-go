package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestRunNotificationServerStopsRuntimeWhenServeFails(t *testing.T) {
	listener := &failingListener{err: errors.New("listener failed")}
	stopCalled := make(chan struct{}, 1)
	cfg := DefaultNotificationServerConfig()
	cfg.HTTP.ShutdownTimeout = time.Second

	err := runNotificationServer(context.Background(), cfg, listener, NotificationServerRuntime{
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		StopRuntime: func(context.Context) error {
			stopCalled <- struct{}{}
			return nil
		},
	}, make(chan os.Signal))
	if err == nil {
		t.Fatal("runNotificationServer() error = nil, want serve failure")
	}
	select {
	case <-stopCalled:
	default:
		t.Fatal("runtime stop was not called after serve failure")
	}
}

func TestBuildCampaignShardWorkersUsesMaxConcurrentShardJobs(t *testing.T) {
	workerIDs := make([]string, 0, 3)
	workers, err := buildCampaignShardWorkers(3, time.Millisecond, func(workerID string) (func(context.Context) error, error) {
		workerIDs = append(workerIDs, workerID)
		return func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}, nil
	})
	if err != nil {
		t.Fatalf("buildCampaignShardWorkers() error = %v", err)
	}
	if len(workers) != 3 {
		t.Fatalf("workers = %d, want 3", len(workers))
	}
	seenWorkerIDs := make(map[string]struct{}, len(workerIDs))
	seenNames := make(map[string]struct{}, len(workers))
	for i, workerID := range workerIDs {
		if workerID == "" {
			t.Fatalf("worker id %d is empty", i)
		}
		if _, ok := seenWorkerIDs[workerID]; ok {
			t.Fatalf("worker id %q reused; each campaign shard worker needs its own lease token", workerID)
		}
		seenWorkerIDs[workerID] = struct{}{}
		name := workers[i].Name()
		if name == "" {
			t.Fatalf("worker name %d is empty", i)
		}
		if _, ok := seenNames[name]; ok {
			t.Fatalf("worker name %q reused; readiness would collapse duplicate workers", name)
		}
		seenNames[name] = struct{}{}
	}
}

type failingListener struct {
	err error
}

func (l *failingListener) Accept() (net.Conn, error) {
	return nil, l.err
}

func (l *failingListener) Close() error {
	return nil
}

func (l *failingListener) Addr() net.Addr {
	return dummyAddr("test")
}

type dummyAddr string

func (a dummyAddr) Network() string {
	return string(a)
}

func (a dummyAddr) String() string {
	return string(a)
}
