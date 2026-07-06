package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
)

type NotificationServerRuntime struct {
	Handler http.Handler
	Closers []namedCloser
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

func newNotificationHTTPServer(cfg NotificationServerConfig, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}
}

func runNotificationServer(ctx context.Context, cfg NotificationServerConfig, listener net.Listener, runtime NotificationServerRuntime, signals <-chan os.Signal) error {
	if listener == nil {
		return fmt.Errorf("notification server listener is required")
	}
	if runtime.Handler == nil {
		return fmt.Errorf("notification server HTTP handler is required")
	}

	server := newNotificationHTTPServer(cfg, runtime.Handler)
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
		return shutdownNotificationServer(ctx, cfg, server, runtime, serveErr)
	case <-ctx.Done():
		return shutdownNotificationServer(ctx, cfg, server, runtime, serveErr)
	}
}

func shutdownNotificationServer(ctx context.Context, cfg NotificationServerConfig, server *http.Server, runtime NotificationServerRuntime, serveErr <-chan error) error {
	// Shutdown owns a fresh drain budget so cancellation from SIGTERM or the
	// parent context does not immediately abort in-flight HTTP requests.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	var errs []error
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

func closeNamedClosers(closers []namedCloser) {
	for _, closer := range closers {
		if closer.closer == nil {
			continue
		}
		_ = closer.closer.Close()
	}
}
