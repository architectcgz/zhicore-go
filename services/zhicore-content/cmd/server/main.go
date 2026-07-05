package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	signals := newShutdownSignalChannel()
	defer signal.Stop(signals)

	err := runContentServerProcess(context.Background(), contentProcessDeps{
		LookupEnv:   os.LookupEnv,
		OpenRuntime: openContentRuntimeDependencies,
		Listen:      net.Listen,
		RunServer:   runContentServer,
		Signals:     signals,
		Log:         os.Stdout,
	})
	if err != nil {
		log.Fatal(err)
	}
}

type contentProcessDeps struct {
	LookupEnv   func(string) (string, bool)
	OpenRuntime func(context.Context, ContentServerConfig) (openedContentRuntime, error)
	Listen      func(network, addr string) (net.Listener, error)
	RunServer   func(context.Context, ContentServerConfig, net.Listener, ContentServerRuntime, <-chan os.Signal) error
	Signals     <-chan os.Signal
	Log         io.Writer
}

func runContentServerProcess(ctx context.Context, deps contentProcessDeps) error {
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.LookupEnv
	}
	if deps.OpenRuntime == nil {
		deps.OpenRuntime = openContentRuntimeDependencies
	}
	if deps.Listen == nil {
		deps.Listen = net.Listen
	}
	if deps.RunServer == nil {
		deps.RunServer = runContentServer
	}
	if deps.Signals == nil {
		deps.Signals = newShutdownSignalChannel()
	}
	if deps.Log == nil {
		deps.Log = io.Discard
	}

	cfg, err := LoadContentServerConfig(deps.LookupEnv)
	if err != nil {
		return fmt.Errorf("load content server config: %w", err)
	}

	opened, err := deps.OpenRuntime(ctx, cfg)
	if err != nil {
		return fmt.Errorf("open content runtime dependencies: %w", err)
	}

	listener, err := deps.Listen("tcp", cfg.HTTP.Addr)
	if err != nil {
		closeNamedClosers(opened.Closers)
		return fmt.Errorf("listen content HTTP server: %w", err)
	}

	fmt.Fprintf(deps.Log, "starting %s\n", cfg.RedactedSummary())
	return deps.RunServer(ctx, cfg, listener, ContentServerRuntime{
		Handler:   opened.Module.HTTPHandler,
		Readiness: opened.Readiness,
		Workers:   opened.Workers,
		Closers:   opened.Closers,
	}, deps.Signals)
}

func newShutdownSignalChannel() chan os.Signal {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	return signals
}
