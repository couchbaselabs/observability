package api

import (
	v1 "github.com/couchbaselabs/observability/config-svc/pkg/api/v1"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) registerRoutes() {
	s.echo.Any("/metrics", echo.WrapHandler(promhttp.Handler()))
	v1.RegisterHandlersWithBaseURL(s.echo, s, "/api/v1")
}
