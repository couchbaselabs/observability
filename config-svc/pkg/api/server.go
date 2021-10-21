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

func NewServer(baseLogger *zap.Logger, configManager metacfg.ConfigManager) (*Server, error) {
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
	server.registerRoutes()
	return &server, nil
}

func (s *Server) Serve() {
	cfg := s.cfg.Get()
	host := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	s.logger.Sugar().Infow("Starting HTTP server", "host", host)
	s.logger.Sugar().Fatalw("HTTP server exited", "err", s.echo.Start(host))
}
