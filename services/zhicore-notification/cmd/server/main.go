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

	err := runNotificationServerProcess(context.Background(), notificationProcessDeps{
		LookupEnv:   os.LookupEnv,
		OpenRuntime: openNotificationRuntimeDependencies,
		Listen:      net.Listen,
		RunServer:   runNotificationServer,
		Signals:     signals,
		Log:         os.Stdout,
	})
	if err != nil {
		log.Fatal(err)
	}
}

type notificationProcessDeps struct {
	LookupEnv   func(string) (string, bool)
	OpenRuntime func(context.Context, NotificationServerConfig) (openedNotificationRuntime, error)
	Listen      func(network, addr string) (net.Listener, error)
	RunServer   func(context.Context, NotificationServerConfig, net.Listener, NotificationServerRuntime, <-chan os.Signal) error
	Signals     <-chan os.Signal
	Log         io.Writer
}

func runNotificationServerProcess(ctx context.Context, deps notificationProcessDeps) error {
	if deps.LookupEnv == nil {
		deps.LookupEnv = os.LookupEnv
	}
	if deps.OpenRuntime == nil {
		deps.OpenRuntime = openNotificationRuntimeDependencies
	}
	if deps.Listen == nil {
		deps.Listen = net.Listen
	}
	if deps.RunServer == nil {
		deps.RunServer = runNotificationServer
	}
	if deps.Signals == nil {
		deps.Signals = newShutdownSignalChannel()
	}
	if deps.Log == nil {
		deps.Log = io.Discard
	}

	cfg, err := LoadNotificationServerConfig(deps.LookupEnv)
	if err != nil {
		return fmt.Errorf("load notification server config: %w", err)
	}
	opened, err := deps.OpenRuntime(ctx, cfg)
	if err != nil {
		return fmt.Errorf("open notification runtime dependencies: %w", err)
	}
	listener, err := deps.Listen("tcp", cfg.HTTP.Addr)
	if err != nil {
		closeNamedClosers(opened.Closers)
		return fmt.Errorf("listen notification HTTP server: %w", err)
	}
	if err := opened.Module.Start(ctx); err != nil {
		closeNamedClosers(opened.Closers)
		_ = listener.Close()
		return fmt.Errorf("start notification runtime module: %w", err)
	}

	fmt.Fprintf(deps.Log, "starting %s\n", cfg.RedactedSummary())
	return deps.RunServer(ctx, cfg, listener, NotificationServerRuntime{
		Handler:     opened.Module.HTTPHandler,
		Closers:     opened.Closers,
		StopRuntime: opened.Module.Stop,
	}, deps.Signals)
}

func newShutdownSignalChannel() chan os.Signal {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	return signals
}
