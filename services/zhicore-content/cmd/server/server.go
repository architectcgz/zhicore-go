package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
)

type ReadinessController interface {
	MarkReady()
	MarkNotReady()
	IsReady() bool
}

type WorkerLifecycle interface {
	Start(context.Context) error
	StopAcceptingNewWork()
	Wait(context.Context) error
}

type ContentServerRuntime struct {
	Handler   http.Handler
	Readiness ReadinessController
	Workers   WorkerLifecycle
	Closers   []namedCloser
}

type namedCloser struct {
	name   string
	closer interface {
		Close() error
	}
}

type closeFunc func() error

func (f closeFunc) Close() error {
	return f()
}

func newContentHTTPServer(cfg ContentServerConfig, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}
}

func runContentServer(ctx context.Context, cfg ContentServerConfig, listener net.Listener, runtime ContentServerRuntime, signals <-chan os.Signal) error {
	if listener == nil {
		return fmt.Errorf("content server listener is required")
	}
	if runtime.Handler == nil {
		return fmt.Errorf("content server HTTP handler is required")
	}

	server := newContentHTTPServer(cfg, runtime.Handler)
	if runtime.Readiness != nil {
		runtime.Readiness.MarkReady()
	}
	if runtime.Workers != nil {
		if err := runtime.Workers.Start(ctx); err != nil {
			if runtime.Readiness != nil {
				runtime.Readiness.MarkNotReady()
			}
			closeNamedClosers(runtime.Closers)
			return fmt.Errorf("start workers: %w", err)
		}
	}

	serveErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErr <- err
	}()

	select {
	case err := <-serveErr:
		closeNamedClosers(runtime.Closers)
		return err
	case <-signals:
		return shutdownContentServer(ctx, cfg, server, runtime, serveErr)
	case <-ctx.Done():
		return shutdownContentServer(ctx, cfg, server, runtime, serveErr)
	}
}

func shutdownContentServer(ctx context.Context, cfg ContentServerConfig, server *http.Server, runtime ContentServerRuntime, serveErr <-chan error) error {
	// A parent cancellation is one shutdown trigger; the graceful shutdown
	// budget still needs its own active context so HTTP requests and workers
	// can drain within the configured timeout instead of being canceled
	// immediately.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	// Readiness flips first so load balancers stop sending traffic before
	// workers claim new async work or dependencies begin closing.
	if runtime.Readiness != nil {
		runtime.Readiness.MarkNotReady()
	}
	if runtime.Workers != nil {
		runtime.Workers.StopAcceptingNewWork()
	}

	var errs []error
	if runtime.Workers != nil {
		if err := runtime.Workers.Wait(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("wait workers: %w", err))
		}
	}
	if err := server.Shutdown(shutdownCtx); err != nil {
		errs = append(errs, fmt.Errorf("shutdown HTTP server: %w", err))
	}
	if err := <-serveErr; err != nil {
		errs = append(errs, fmt.Errorf("serve HTTP: %w", err))
	}
	for _, closer := range runtime.Closers {
		if closer.closer == nil {
			continue
		}
		if err := closer.closer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close %s: %w", closer.name, err))
		}
	}

	return errors.Join(errs...)
}
