package manager

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"net/http"
	"time"

	"github.com/couchbaselabs/cbmultimanager/configuration"
	"github.com/couchbaselabs/cbmultimanager/heart"
	"github.com/couchbaselabs/cbmultimanager/janitor"
	"github.com/couchbaselabs/cbmultimanager/statistics"
	"github.com/couchbaselabs/cbmultimanager/status"
	"github.com/couchbaselabs/cbmultimanager/storage"
	"github.com/couchbaselabs/cbmultimanager/storage/sqlite"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/pbkdf2"
)

// FrequencyConfiguration is just a convenient grouping of all the frequencies for the different monitors the manger
// runs.
type FrequencyConfiguration struct {
	Heart   time.Duration
	Status  time.Duration
	Janitor time.Duration
}

// DefaultFrequencyConfiguration is the default frequencies used.
var DefaultFrequencyConfiguration = FrequencyConfiguration{
	Heart:   time.Minute,
	Status:  5 * time.Minute,
	Janitor: 6 * time.Hour,
}

// Manager is the struct in charge of running the various monitors as well as the REST endpoints.
type Manager struct {
	config *configuration.Config

	store         storage.Store
	heartMonitor  heart.MonitorIFace
	statusMonitor status.MonitorInterface
	janitor       *janitor.Janitor

	initialized bool

	ctx    context.Context
	cancel context.CancelFunc

	httpServer  *http.Server
	httpsServer *http.Server
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

	return &Manager{
		config:        config,
		store:         store,
		initialized:   initialized,
		heartMonitor:  heart.NewMonitor(store, config.MaxWorkers),
		statusMonitor: status.NewMonitor(store, config.MaxWorkers),
		janitor:       janitor.NewJanitor(store),
	}, nil
}

// Start will start all the monitors as well as the rest server.
func (m *Manager) Start(config FrequencyConfiguration) {
	if m.ctx != nil {
		return
	}

	zap.S().Infow("(Manager) Starting", "frequencies", config)
	m.ctx, m.cancel = context.WithCancel(context.Background())

	m.setupKeys()
	statistics.CollectStats()

	m.startRESTServers()
	m.heartMonitor.Start(config.Heart)
	m.statusMonitor.Start(config.Status)
	m.janitor.Start(config.Janitor)

	zap.S().Info("(Manager) Started")
	<-m.ctx.Done()
	zap.S().Info("(Manger) Stopped")
}

// Stop stops the manager and all its monitors.
func (m *Manager) Stop() {
	if m.ctx == nil {
		return
	}

	zap.S().Info("(Manger) Stopping")
	m.heartMonitor.Stop()
	m.statusMonitor.Stop()
	m.janitor.Stop()

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
