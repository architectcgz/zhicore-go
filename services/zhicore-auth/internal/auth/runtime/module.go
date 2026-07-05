package runtime

import (
	"fmt"
	"net/http"

	authhttp "github.com/architectcgz/zhicore-go/services/zhicore-auth/api/http"
	"github.com/gin-gonic/gin"
)

type Deps struct {
	Service authhttp.Service
}

type Module struct {
	HTTPHandler *gin.Engine
}

func Build(deps Deps) (*Module, error) {
	if deps.Service == nil {
		return nil, fmt.Errorf("auth runtime Service dependency is required")
	}

	root := authhttp.NewHandler(deps.Service)
	root.GET("/health/live", healthHandler())
	root.GET("/health/ready", healthHandler())

	return &Module{
		HTTPHandler: root,
	}, nil
}

func healthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Readiness stays dependency-free in this slice because repository/Redis/MQ adapters are not wired yet.
		// Once runtime owns real downstream clients, this handler should reflect required dependency health.
		c.String(http.StatusOK, "ok")
	}
}
