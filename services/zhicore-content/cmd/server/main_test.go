package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"

	contentruntime "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/runtime"
	"github.com/gin-gonic/gin"
)

func TestContentServerProcessLoadsConfigOpensDepsAndRunsServer(t *testing.T) {
	var openedCfg ContentServerConfig
	var listenedAddr string
	var ranCfg ContentServerConfig
	var log bytes.Buffer
	listener := stubListener{addr: stubAddr("127.0.0.1:19080")}
	signals := make(chan os.Signal, 1)
	deps := contentProcessDeps{
		LookupEnv: mapLookup(sensitiveConfigValues()),
		OpenRuntime: func(ctx context.Context, cfg ContentServerConfig) (openedContentRuntime, error) {
			if ctx == nil {
				t.Fatal("OpenRuntime ctx = nil")
			}
			openedCfg = cfg
			return openedContentRuntime{
				Module: &contentruntime.Module{
					HTTPHandler: gin.New(),
				},
				Readiness: &recordingReadiness{},
			}, nil
		},
		Listen: func(network, addr string) (net.Listener, error) {
			if network != "tcp" {
				t.Fatalf("Listen network = %q, want tcp", network)
			}
			listenedAddr = addr
			return listener, nil
		},
		RunServer: func(ctx context.Context, cfg ContentServerConfig, gotListener net.Listener, runtime ContentServerRuntime, gotSignals <-chan os.Signal) error {
			if ctx == nil {
				t.Fatal("RunServer ctx = nil")
			}
			if gotListener != listener {
				t.Fatal("RunServer listener did not receive opened listener")
			}
			if runtime.Handler == nil || runtime.Readiness == nil {
				t.Fatalf("RunServer runtime = %#v, want handler and readiness", runtime)
			}
			if gotSignals != signals {
				t.Fatal("RunServer signals did not receive configured signal channel")
			}
			ranCfg = cfg
			return nil
		},
		Signals: signals,
		Log:     &log,
	}

	if err := runContentServerProcess(context.Background(), deps); err != nil {
		t.Fatalf("runContentServerProcess() error = %v", err)
	}

	if openedCfg.HTTP.Addr != ":19080" {
		t.Fatalf("opened config HTTP.Addr = %q, want :19080", openedCfg.HTTP.Addr)
	}
	if listenedAddr != ":19080" {
		t.Fatalf("listened addr = %q, want configured addr", listenedAddr)
	}
	if ranCfg.HTTP.Addr != ":19080" {
		t.Fatalf("run config HTTP.Addr = %q, want :19080", ranCfg.HTTP.Addr)
	}
	assertNoSensitiveConfigLeak(t, log.String())
	if !strings.Contains(log.String(), "zhicore-content") || !strings.Contains(log.String(), "http.addr=:19080") {
		t.Fatalf("log = %q, want redacted startup summary", log.String())
	}
}

func TestContentServerProcessClosesOpenedDepsWhenListenFails(t *testing.T) {
	var closed bool
	deps := contentProcessDeps{
		LookupEnv: mapLookup(validRequiredConfigValues()),
		OpenRuntime: func(context.Context, ContentServerConfig) (openedContentRuntime, error) {
			return openedContentRuntime{
				Module: &contentruntime.Module{HTTPHandler: gin.New()},
				Closers: []namedCloser{
					{name: "postgres", closer: closeFunc(func() error {
						closed = true
						return nil
					})},
				},
				Readiness: &recordingReadiness{},
			}, nil
		},
		Listen: func(string, string) (net.Listener, error) {
			return nil, errors.New("listen failed")
		},
		RunServer: func(context.Context, ContentServerConfig, net.Listener, ContentServerRuntime, <-chan os.Signal) error {
			t.Fatal("RunServer should not be called when Listen fails")
			return nil
		},
		Signals: make(chan os.Signal),
	}

	err := runContentServerProcess(context.Background(), deps)
	if err == nil || !strings.Contains(err.Error(), "listen content HTTP server") {
		t.Fatalf("runContentServerProcess() error = %v, want listen failure", err)
	}
	if !closed {
		t.Fatal("opened dependencies were not closed after listen failure")
	}
}

func TestContentServerProcessRejectsInvalidConfigBeforeOpeningDeps(t *testing.T) {
	deps := contentProcessDeps{
		LookupEnv: mapLookup(nil),
		OpenRuntime: func(context.Context, ContentServerConfig) (openedContentRuntime, error) {
			t.Fatal("OpenRuntime should not be called after config validation failure")
			return openedContentRuntime{}, nil
		},
		Listen: func(string, string) (net.Listener, error) {
			t.Fatal("Listen should not be called after config validation failure")
			return nil, nil
		},
		Signals: make(chan os.Signal),
	}

	err := runContentServerProcess(context.Background(), deps)
	if err == nil || !strings.Contains(err.Error(), "load content server config") {
		t.Fatalf("runContentServerProcess() error = %v, want config load failure", err)
	}
}

func TestContentServerMainSignalsIncludeInterruptAndTerminate(t *testing.T) {
	signals := newShutdownSignalChannel()
	defer close(signals)

	signals <- syscall.SIGINT
	if got := <-signals; got != syscall.SIGINT {
		t.Fatalf("signal = %v, want SIGINT", got)
	}
	signals <- syscall.SIGTERM
	if got := <-signals; got != syscall.SIGTERM {
		t.Fatalf("signal = %v, want SIGTERM", got)
	}
}

type stubListener struct {
	net.Listener
	addr net.Addr
}

func (l stubListener) Addr() net.Addr { return l.addr }

func (l stubListener) Close() error { return nil }

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }

func (a stubAddr) String() string { return string(a) }
