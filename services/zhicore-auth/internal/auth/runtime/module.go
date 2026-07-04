package runtime

import (
	"fmt"
	"net/http"

	authhttp "github.com/architectcgz/zhicore-go/services/zhicore-auth/api/http"
)

type Deps struct {
	Service authhttp.Service
}

type Module struct {
	HTTPHandler  http.Handler
	LiveHandler  http.Handler
	ReadyHandler http.Handler
}

func Build(deps Deps) (*Module, error) {
	if deps.Service == nil {
		return nil, fmt.Errorf("auth runtime Service dependency is required")
	}

	liveHandler := healthHandler()
	readyHandler := healthHandler()
	authHandler := authhttp.NewHandler(deps.Service)

	root := http.NewServeMux()
	root.Handle("GET /health/live", liveHandler)
	root.Handle("GET /health/ready", readyHandler)
	root.Handle("/api/v1/auth/", authHandler)

	return &Module{
		HTTPHandler:  root,
		LiveHandler:  liveHandler,
		ReadyHandler: readyHandler,
	}, nil
}

func healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Readiness stays dependency-free in this slice because repository/Redis/MQ adapters are not wired yet.
		// Once runtime owns real downstream clients, this handler should reflect required dependency health.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}
