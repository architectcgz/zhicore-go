package runtime

import (
	"fmt"
	"net/http"

	userhttp "github.com/architectcgz/zhicore-go/services/zhicore-user/api/http"
)

type Deps struct {
	Service           userhttp.Service
	AvatarURLResolver userhttp.AvatarURLResolver
}

type Module struct {
	HTTPHandler  http.Handler
	LiveHandler  http.Handler
	ReadyHandler http.Handler
}

func Build(deps Deps) (*Module, error) {
	if deps.Service == nil {
		return nil, fmt.Errorf("user runtime Service dependency is required")
	}

	liveHandler := healthHandler()
	readyHandler := healthHandler()
	userHandler := userhttp.NewHandler(deps.Service, deps.AvatarURLResolver)

	root := http.NewServeMux()
	root.Handle("GET /health/live", liveHandler)
	root.Handle("GET /health/ready", readyHandler)
	root.Handle("/api/v1/users/", userHandler)

	return &Module{
		HTTPHandler:  root,
		LiveHandler:  liveHandler,
		ReadyHandler: readyHandler,
	}, nil
}

func healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Readiness remains dependency-free until User owns real repository, cache,
		// File client and outbox adapters; runtime must not fake downstream health.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}
