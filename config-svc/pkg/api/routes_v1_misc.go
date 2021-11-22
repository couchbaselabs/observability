package api

import (
	"net/http"

	v1 "github.com/couchbaselabs/observability/config-svc/pkg/api/v1"

	"github.com/labstack/echo/v4"
)

func (s *Server) GetOpenapiJson(ctx echo.Context) error { //nolint:revive
	swagger, err := v1.GetSwagger()
	if err != nil {
		return err
	}
	return ctx.JSONPretty(http.StatusOK, swagger, "\t")
}
