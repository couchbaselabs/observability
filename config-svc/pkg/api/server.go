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

	"github.com/brpaz/echozap"
	"github.com/couchbaselabs/observability/config-svc/pkg/manager"
	"github.com/couchbaselabs/observability/config-svc/pkg/metacfg"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type Server struct {
	baseLogger *zap.Logger
	logger     *zap.Logger
	cfg        metacfg.ConfigManager
	clusters   *manager.ClusterManager
	echo       *echo.Echo
}

func NewServer(baseLogger *zap.Logger, configManager metacfg.ConfigManager, pathPrefix string) (*Server, error) {
	mgr, err := manager.NewClusterManager(baseLogger, configManager)
	if err != nil {
		return nil, err
	}
	server := Server{
		baseLogger: baseLogger,
		cfg:        configManager,
		logger:     baseLogger.Named("server"),
		echo:       echo.New(),
		clusters:   mgr,
	}
	server.echo.HideBanner = true
	server.echo.HidePort = true
	server.echo.Use(echozap.ZapLogger(server.logger))
	server.registerRoutes(pathPrefix)
	return &server, nil
}

func (s *Server) Serve(host string, port int) {
	go s.clusters.StartUpdating()
	listenHost := fmt.Sprintf("%s:%d", host, port)
	s.logger.Sugar().Infow("Starting HTTP server", "host", host)
	s.logger.Sugar().Fatalw("HTTP server exited", "err", s.echo.Start(listenHost))
}
