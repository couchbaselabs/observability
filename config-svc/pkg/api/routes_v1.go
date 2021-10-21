package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func (s *Server) GetConfig(ctx echo.Context) error {
	return ctx.Blob(http.StatusOK, "text/yaml", s.cfg.Get().ToYAML())
}

func (s *Server) GetClusters(ctx echo.Context) error {
	return nil
}
