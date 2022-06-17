// Copyright 2021 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the  License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4/middleware"

	"github.com/brpaz/echozap"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type Server struct {
	baseLogger *zap.Logger
	logger     *zap.Logger
	echo       *echo.Echo
	production bool
}

func NewServer(baseLogger *zap.Logger, pathPrefix string, production bool) (*Server, error) {
	server := Server{
		baseLogger: baseLogger,
		logger:     baseLogger.Named("server"),
		echo:       echo.New(),
		production: production,
	}
	server.echo.HideBanner = true
	server.echo.HidePort = true
	server.echo.Use(echozap.ZapLogger(server.logger))
	server.echo.Use(middleware.Recover())
	server.echo.Use(middleware.CORS())
	server.echo.HTTPErrorHandler = server.handleError
	server.registerRoutes(pathPrefix)
	return &server, nil
}

func (s *Server) handleError(err error, ctx echo.Context) {
	code := http.StatusInternalServerError
	msg := err.Error()
	if httpErr, ok := err.(*echo.HTTPError); ok {
		code = httpErr.Code
		msg = fmt.Sprintf("%v", httpErr.Message)
	}
	_ = ctx.JSON(code, map[string]interface{}{
		"ok":  false,
		"err": msg,
	})
}

func (s *Server) Serve(host string, port int) {
	listenHost := fmt.Sprintf("%s:%d", host, port)
	s.logger.Sugar().Infow("Starting HTTP server", "host", listenHost)
	s.logger.Sugar().Fatalw("HTTP server exited", "err", s.echo.Start(listenHost))
}
