package api

import (
	v1 "github.com/couchbaselabs/observability/config-svc/pkg/api/v1"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (s *Server) GetConfig(ctx echo.Context) error {
	return ctx.Blob(http.StatusOK, "text/yaml", s.cfg.Get().ToYAML())
}

func (s *Server) GetClusters(ctx echo.Context) error {
	return nil
}

func (s *Server) GetOpenapiJson(ctx echo.Context) error {
	swagger, err := v1.GetSwagger()
	if err != nil {
		return err
	}
	return ctx.JSONPretty(http.StatusOK, swagger, "\t")
}
