// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/bootstrap"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/config"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/fluentbit"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/hazelnut"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/handlers"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/janitor"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/runner"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/store"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/prometheus"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/server"
)

type Agent struct {
	state    AgentState
	logger   *zap.SugaredLogger
	cfg      config.Config
	features config.FeatureSet

	store     *store.InMemory
	server    *server.Server
	exporter  *prometheus.Exporter
	runner    *runner.Runner
	fluentBit *fluentbit.FluentBit
	hazelnut  *hazelnut.Receiver
	janitor   *janitor.Janitor

	node     *bootstrap.Node
	baseCtx  context.Context
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	serverWg sync.WaitGroup
}

func CreateAgent(cfg config.Config) (*Agent, error) {
	return &Agent{
		state:    AgentNotStarted,
		cfg:      cfg,
		features: config.NoFeatures(),
		logger:   zap.S().Named("Agent Core"),
	}, nil
}

const _credsFileName = "cbhealthagent.pw"

func (a *Agent) Start(baseContext context.Context) error {
	a.baseCtx = baseContext

	// If the user has directly supplied credentials, always use those - if they're invalid, fail-fast rather than
	// get stuck waiting for a cluster monitor that may never appear.
	if a.cfg.CouchbaseUsername != "" && a.cfg.CouchbasePassword != "" {
		bootstrapper := bootstrap.NewKnownCredentialsBootstrapper(a.cfg.CouchbaseUsername, a.cfg.CouchbasePassword)
		var err error
		a.node, err = bootstrapper.CreateRESTClient()
		if err != nil {
			return fmt.Errorf("failed to bootstrap using supplied cluster credentials: %w", err)
		}
		a.state = AgentReady
		return a.startup()
	}

	credsFile, err := a.getCredentialsFilePath()
	if err != nil {
		return fmt.Errorf("failed to determine credentials path: %w", err)
	}
	creds, err := readCredentialsFromFile(credsFile)
	if errors.Is(err, os.ErrNotExist) {
		a.logger.Infof("Cached credentials not found at %s, "+
			"now waiting to receive credentials from a cluster monitor.", credsFile)
		a.state = AgentWaiting
		return a.startup()
	}
	if err != nil {
		return fmt.Errorf("could not read credentials: %w", err)
	}

	// Performing this bootstrap will also verify that the loaded credentials are still valid.
	bootstrapper := bootstrap.NewKnownCredentialsBootstrapper(creds.GetCredentials("localhost"))
	a.node, err = bootstrapper.CreateRESTClient()
	var bootstrapErr *cbrest.BootstrapFailureError
	if errors.As(err, &bootstrapErr) {
		if bootstrapErr.ErrAuthorization != nil || bootstrapErr.ErrAuthentication != nil {
			a.logger.Infof("Found invalid cached credentials at %s, "+
				"now waiting to receive credentials from a cluster monitor.", credsFile)
			a.state = AgentWaiting
			return a.startup()
		}
	}
	if err != nil {
		return fmt.Errorf("failed to bootstrap: %w", err)
	}

	a.state = AgentReady
	return a.startup()
}

// Done returns a channel that will never receive any values, and will be closed
// once all the services this agent is managing have fully shut down.
func (a *Agent) Done() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(ch)
	}()
	return ch
}

// Shutdown brings down the entire agent, including all services as well as the HTTP server.
func (a *Agent) Shutdown() {
	credsFile, err := a.getCredentialsFilePath()
	if err != nil {
		a.logger.Infow("Failed to determine credentials path for purge", "error", err)
	} else {
		if err = removeCredentialsFile(credsFile); err != nil {
			a.logger.Infow("Unable to remove credentials file", "error", err)
		}
	}
	a.shutdownServices()
	if a.server != nil {
		a.server.Close()
		a.serverWg.Wait()
	}
}

// shutdownServices brings down the server's services, but not the HTTP server.
func (a *Agent) shutdownServices() {
	if a.exporter != nil {
		a.exporter.Shutdown()
	}
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()
}

func (a *Agent) startup() error {
	// NOTE: server uses baseCtx, not ctx, to ensure that it doesn't
	// get brought down if shutdownServices is called (e.g. when a waiting agent gets activated)
	a.ctx, a.cancel = context.WithCancel(a.baseCtx)

	active := a.node != nil

	var err error
	if active {
		a.features, err = config.DetermineFeaturesForNode(a.node,
			a.cfg.AutoFeatures,
			a.cfg.EnableFeatures,
			a.cfg.DisableFeatures,
		)
	} else {
		a.features, err = config.DetermineFeaturesBasedOnFlags(
			a.cfg.AutoFeatures,
			a.cfg.EnableFeatures,
			a.cfg.DisableFeatures,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to determine features: %w", err)
	}

	a.store = store.NewInMemoryStore()
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(AgentActiveHeader, fmt.Sprintf("%t", active))
		http.NotFound(w, r)
	})
	a.registerRoutes(router)

	if active { //nolint:nestif
		if a.features[config.FeatHealthAgent] {
			a.logger.Info("Starting health checks")
			a.runner = runner.NewRunner(a.node, a.cfg.CheckInterval, a.store)
			a.wg.Add(1)
			a.runner.Start(a.ctx, &a.wg)
			handlers.RegisterHealthAgentRoutes(router, a.store)
		} else {
			a.logger.Info("Health agent disabled.")
		}

		if a.features[config.FeatLogAnalyzer] {
			a.logger.Info("Starting log analyzer")
			a.hazelnut, err = hazelnut.NewHazelnut(a.store, a.node)
			if err != nil {
				return fmt.Errorf("failed to create log analyzer: %w", err)
			}
			a.wg.Add(1)
			err = a.hazelnut.Start(a.ctx, &a.wg)
			if err != nil {
				return fmt.Errorf("failed to start log analyzer: %w", err)
			}
		} else {
			a.logger.Info("Log analyzer disabled.")
		}

		if a.features[config.FeatFluentBit] {
			// Needs to be started after Hazelnut as it needs to know the port
			var hzPort int
			if a.hazelnut != nil {
				hzPort = a.hazelnut.Port()
			}
			a.logger.Info("Starting Fluent Bit")
			a.fluentBit, err = fluentbit.NewFluentBit(a.node, a.cfg.CouchbaseInstallPath, hzPort)
			if err != nil {
				return fmt.Errorf("failed to create Fluent Bit: %w", err)
			}
			a.wg.Add(1)
			err = a.fluentBit.Start(a.ctx, &a.wg)
			if err != nil {
				return fmt.Errorf("failed to start Fluent Bit: %w", err)
			}
		} else {
			a.logger.Info("Fluent Bit disabled.")
		}
	}

	// Prometheus exporter is always brought up, however if we're in Waiting state we won't bring up all the services

	if a.features[config.FeatHealthAgent] || a.features[config.FeatLogAnalyzer] {
		zap.S().Info("Starting janitor")
		a.janitor = janitor.NewJanitor(a.store, a.cfg.JanitorInterval, a.cfg.LogAlertDuration)
		a.wg.Add(1)
		a.janitor.Start(a.ctx, &a.wg)
	}

	if a.features[config.FeatPrometheusExporter] {
		a.logger.Info("Starting Prometheus exporter")
		a.exporter, err = prometheus.NewExporter(a.node)
		if err != nil {
			return fmt.Errorf("failed to create Prometheus exporter: %w", err)
		}
		a.exporter.Register(router)
	} else {
		a.logger.Info("Prometheus exporter disabled.")
	}

	if a.server == nil {
		a.logger.Debug("Starting HTTP server")
		a.serverWg.Add(1)
		a.server = server.NewServer(a.store, a.cfg.HTTPPort, router)
		if err := a.server.Start(a.baseCtx, &a.serverWg); err != nil {
			return fmt.Errorf("failed to start HTTP server: %w", err)
		}
	} else {
		a.server.ReplaceRouter(router)
	}

	a.logger.Info("Startup complete.")
	return nil
}

func (a *Agent) getCredentialsFilePath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not determine path to cbhealthagent executable: %w", err)
	}
	credsFile := filepath.Join(filepath.Dir(executable), _credsFileName)
	return credsFile, nil
}
