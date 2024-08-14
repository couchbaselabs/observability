// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/auth"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/configuration"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/discovery"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/discovery/prometheus"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/janitor"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/statistics"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/status"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/status/alertmanager"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage/sqlite"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/pbkdf2"
)

// DefaultFrequencyConfiguration is the default frequencies used.
var DefaultFrequencyConfiguration = values.FrequencyConfiguration{
	Heart:                time.Minute,
	Status:               5 * time.Minute,
	Janitor:              6 * time.Hour,
	DiscoveryRunsTimeGap: time.Minute,
	AgentPortReconcile:   2 * time.Minute,
}

// Manager is the struct in charge of running the various monitors as well as the REST endpoints.
// Responsible for spawning the SingleClusterManagers, as well as serving the API.
type Manager struct {
	config *configuration.Config
	client *couchbase.Client

	store            storage.Store
	discoveryManager discovery.Manager
	janitor          *janitor.Janitor
	alertmanager     *alertmanager.AlertGenerator
	checkExecutor    *status.CheckExecutor

	clusterManagers ClusterManagers

	initialized bool

	ctx    context.Context
	cancel context.CancelFunc

	httpServer  *http.Server
	httpsServer *http.Server
}

type ClusterManagers struct {
	cms map[string]ClusterManager
	mx  sync.RWMutex
}

func NewDefaultClusterManagers() ClusterManagers {
	return ClusterManagers{
		cms: make(map[string]ClusterManager),
	}
}

func NewClusterManagers(cms map[string]ClusterManager) ClusterManagers {
	return ClusterManagers{cms: cms}
}

func (c *ClusterManagers) Store(k string, v ClusterManager) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	// won't amend; only add
	if _, ok := c.cms[k]; !ok {
		c.cms[k] = v
	}
}

func (c *ClusterManagers) Load(k string) (v ClusterManager, ok bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	v, ok = c.cms[k]
	return v, ok
}

func (c *ClusterManagers) Remove(k string) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	delete(c.cms, k)
}

// NewManager creates a new manager with the store and all monitors initialized.
func NewManager(config *configuration.Config) (*Manager, error) {
	store, err := sqlite.NewSQLiteDB(config.SQLiteDB, config.SQLiteKey)
	if err != nil {
		return nil, fmt.Errorf("could not initialize store: %w", err)
	}

	initialized, err := store.IsInitialized()
	if err != nil {
		return nil, fmt.Errorf("could not determine state of store: %w", err)
	}

	manager := Manager{
		config:      config,
		client:      nil,
		store:       store,
		initialized: initialized,
		janitor: janitor.NewJanitor(store, janitor.Config{
			LogAlertMaxAge: config.LogCheckLifetime,
		}),
		checkExecutor:   status.NewCheckExecutor(config.MaxWorkers),
		clusterManagers: NewDefaultClusterManagers(),
	}

	if config.AdminPassword != "" {
		hashedPassword, err := auth.HashPassword(config.AdminPassword)
		if err != nil {
			return nil, fmt.Errorf("could not hash password: %w", err)
		}

		if config.AdminUser != "" {
			err = manager.SetupAdminUser(config.AdminUser, hashedPassword)
			if err != nil {
				return nil, fmt.Errorf("could not create admin user: %w", err)
			}
		} else {
			return nil, fmt.Errorf("admin password set but no admin user")
		}
	}

	if config.PrometheusBaseURL != "" && config.PrometheusLabelSelector != nil {
		// TODO (CMOS-58) make this more generic
		promDiscovery, err := prometheus.NewPrometheusCouchbaseClusterDiscovery(config, store)
		if err != nil {
			return nil, fmt.Errorf("could not create Prometheus discovery: %w", err)
		}
		manager.discoveryManager, err = discovery.NewClusterDiscoveryManager(promDiscovery)
		if err != nil {
			return nil, fmt.Errorf("could not create discovery manager: %w", err)
		}
	}

	if len(config.AlertmanagerURLs) > 0 {
		manager.alertmanager = alertmanager.NewAlertGenerator(store, config.AlertmanagerResendDelay, config.AlertmanagerURLs,
			config.AlertmanagerBaseLabels)
	}

	return &manager, nil
}

// Start will start all the monitors as well as the rest server.
func (m *Manager) Start(config values.FrequencyConfiguration) {
	if m.ctx != nil {
		return
	}

	// map keeping track of all started scms.
	scmStarted := make(map[string]struct{})

	zap.S().Infow("(Manager) Starting", "frequencies", config)
	m.ctx, m.cancel = context.WithCancel(context.Background())

	m.setupKeys()
	statistics.RegisterStatsCollection()

	m.startRESTServers()
	m.janitor.Start(config.Janitor)

	zap.S().Info("(Manager) Starting..")

	if m.alertmanager != nil {
		m.alertmanager.Start()
	}

	// mostly for already added(manually)/discovered(auto-discovered via Prom) from SQLLite DB, between runs.
	// for manually added clusters, cluster managers are started while addition.
	// for auto-discovered(via Prom) clusters, cluster managers are started while discovered via discoverLoop.
	if _, err := m.store.GetClusters(true, false); err == nil {
		err := m.populateClusterManagersForAllClustersFromStore()
		if err != nil {
			zap.S().Warnw("(Manager) Error populating clusterManagers: %w", err)
		}

		for cuuid, cm := range m.clusterManagers.cms {
			if _, ok := scmStarted[cuuid]; !ok {
				err := cm.Start()
				if err != nil {
					zap.S().Errorw("(Manager) Error startin single cluster manager: %w", err)
					continue
				}
				// adds started scm uuid to map.
				scmStarted[cuuid] = struct{}{}
			}
		}
	}

	if m.discoveryManager != nil {
		m.discoveryManager.Start(config.DiscoveryRunsTimeGap)
		for {
			select {
			case status := <-m.discoveryManager.HasBeenDiscovered():
				switch status {
				case discovery.ClusterDiscoveryStatusSuccess:
					err := m.populateClusterManagersForAllClustersFromStore()
					if err != nil {
						zap.S().Warnw("(Manager) Error populating clusterManagers: %w", err)
					}
					for cuuid, cm := range m.clusterManagers.cms {
						if _, ok := scmStarted[cuuid]; !ok {
							err := cm.Start()
							if err != nil {
								zap.S().Errorw("(Manager) Failed to start Single Cluster Manager: %w", err)
								continue
							}
							// adds started scm uuid to map.
							scmStarted[cuuid] = struct{}{}
						}
					}
				case discovery.ClusterDiscoveryStatusFailure:
					zap.S().Warnw("(Manager) Error discovering clusters:")
					continue
				}

			case <-m.ctx.Done():
				return
			}
		}
	}

	<-m.ctx.Done()
	zap.S().Info("(Manager) Stopping..")
}

// Stop stops the manager and all its monitors.
func (m *Manager) Stop() {
	if m.ctx == nil {
		return
	}

	statistics.UnregisterStatsCollection()

	zap.S().Info("(Manager) Stopping")
	for _, cm := range m.clusterManagers.cms {
		cm.Stop()
	}
	if err := m.checkExecutor.Stop(); err != nil {
		zap.S().Warnw("(Manager) Error when stopping check executor", "error", err)
	}
	m.janitor.Stop()
	if m.discoveryManager != nil {
		m.discoveryManager.Stop()
	}
	if m.alertmanager != nil {
		m.alertmanager.Stop()
	}

	m.stopRESTServers()

	m.cancel()
	m.ctx, m.cancel = nil, nil
}

func (m *Manager) stopRESTServers() {
	zap.S().Infow("(Manager) Stopping REST servers")
	if m.httpServer != nil {
		_ = m.httpServer.Shutdown(context.Background())
	}

	if m.httpsServer != nil {
		_ = m.httpsServer.Shutdown(context.Background())
	}

	m.httpServer, m.httpsServer = nil, nil
}

func (m *Manager) startRESTServers() {
	r := NewRouter(m)

	if !m.config.DisableHTTP {
		go func() {
			m.httpServer = &http.Server{
				Addr:    fmt.Sprintf(":%d", m.config.HTTPPort),
				Handler: r,
			}

			zap.S().Infow("(Manager) (HTTP) Starting HTTP server", "port", m.config.HTTPPort)
			if err := m.httpServer.ListenAndServe(); err != nil {
				zap.S().Warnw("(Manager) (HTTP) Server stopped", "err", err)
			}
		}()
	}

	if !m.config.DisableHTTPS {
		if m.config.CertPath == "" || m.config.KeyPath == "" {
			zap.S().Warn("(Manager) TLS certificate/key not set, TLS disabled.")
		} else {
			go func() {
				m.httpsServer = &http.Server{
					Addr:    fmt.Sprintf(":%d", m.config.HTTPSPort),
					Handler: r,
				}

				zap.S().Infow("(Manager) (HTTPS) Starting HTTPS server", "port", m.config.HTTPSPort)
				if err := m.httpsServer.ListenAndServeTLS(m.config.CertPath, m.config.KeyPath); err != nil {
					zap.S().Warnw("(Manager) (HTTPS) Server stopped", "err", err)
				}
			}()
		}
	}
}

// setupKeys generates the keys it will use to encrypt and sign the JWTs. This keys are derived from the SQLite key so
// the user only has to pass one key.
//
// NOTE: As this keys are derived on start up and are volatile. This means that if cbmultimanager is restarted all the
// tokens are invalidated. This is not a big deal it means that users will have to sign in again. This would happen
// regardless as tokens expire after one hour.
func (m *Manager) setupKeys() {
	// set a UUID this will be used for tokens and some other stuff
	m.config.UUID = uuid.New().String()

	salt := make([]byte, 16)
	// safe to ignore always returns nil
	_, _ = rand.Read(salt)
	// we will use this key for JWT so it can be ephemeral. We use AES256 which requires a 256 bit (32 byte) key
	// if we ever make this into a distributed system this key will have to be shared by all nodes in which case is
	// best if provided by the user.
	m.config.EncryptKey = pbkdf2.Key([]byte(m.config.SQLiteKey), salt, 4096, 32, sha512.New)
	m.config.SignKey = pbkdf2.Key([]byte(m.config.SQLiteKey), salt, 4096, 64, sha512.New)
}

func (m *Manager) populateClusterManagersForAllClustersFromStore() error {
	// for all clusters from store including sensitive fields
	clustersFromStore, err := m.store.GetClusters(true, false)
	if err != nil {
		return fmt.Errorf("could not get clusters: %w", err)
	}

	for _, clusterFrmStore := range clustersFromStore {
		m.clusterManagers.Store(clusterFrmStore.UUID, NewSingleClusterManager(
			clusterFrmStore,
			m.client,
			m.store,
			m.alertmanager,
			m.checkExecutor,
			DefaultFrequencyConfiguration,
		))
	}

	return nil
}
