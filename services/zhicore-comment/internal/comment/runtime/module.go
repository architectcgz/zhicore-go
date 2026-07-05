package runtime

import (
	"context"
	"fmt"
	"net/http"

	commenthttp "github.com/architectcgz/zhicore-go/services/zhicore-comment/api/http"
)

type Worker interface {
	Run(context.Context) error
}

type Deps struct {
	Service commenthttp.Service
	Workers []Worker
}

type Module struct {
	HTTPHandler  http.Handler
	LiveHandler  http.Handler
	ReadyHandler http.Handler
	Workers      []Worker
}

func Build(deps Deps) (*Module, error) {
	if deps.Service == nil {
		return nil, fmt.Errorf("comment runtime Service dependency is required")
	}

	liveHandler := healthHandler()
	readyHandler := healthHandler()
	commentHandler := commenthttp.NewHandler(deps.Service)

	root := http.NewServeMux()
	root.Handle("GET /health/live", liveHandler)
	root.Handle("GET /health/ready", readyHandler)
	root.Handle("/api/v1/posts/", commentHandler)
	root.Handle("/api/v1/admin/comments/", commentHandler)

	return &Module{
		HTTPHandler:  root,
		LiveHandler:  liveHandler,
		ReadyHandler: readyHandler,
		Workers:      append([]Worker(nil), deps.Workers...),
	}, nil
}

func healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Readiness stays dependency-free until PostgreSQL, RabbitMQ, Redis and
		// downstream clients have concrete runtime adapters wired.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}
