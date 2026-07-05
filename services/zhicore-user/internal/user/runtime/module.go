package runtime

import (
	"fmt"
	"net/http"

	userhttp "github.com/architectcgz/zhicore-go/services/zhicore-user/api/http"
	"github.com/gin-gonic/gin"
)

type Deps struct {
	Service           userhttp.Service
	AvatarURLResolver userhttp.AvatarURLResolver
}

type Module struct {
	HTTPHandler *gin.Engine
}

func Build(deps Deps) (*Module, error) {
	if deps.Service == nil {
		return nil, fmt.Errorf("user runtime Service dependency is required")
	}

	root := userhttp.NewHandler(deps.Service, deps.AvatarURLResolver)
	root.GET("/health/live", healthHandler())
	root.GET("/health/ready", healthHandler())

	return &Module{
		HTTPHandler: root,
	}, nil
}

func healthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Readiness remains dependency-free until User owns real repository, cache,
		// File client and outbox adapters; runtime must not fake downstream health.
		c.String(http.StatusOK, "ok")
	}
}
