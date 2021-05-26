package status

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/couchbaselabs/cbmultimanager/status/progress"
	"github.com/couchbaselabs/cbmultimanager/storage"
	"github.com/couchbaselabs/cbmultimanager/values"

	"go.uber.org/zap"
)

// apiMonitor is in charge of running the checkers that are based on the Couchbase Cluster REST API. In the future we
// may have other type of monitors.
type apiMonitor struct {
	store storage.Store

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// distribute workers
	workerWg     *sync.WaitGroup
	workerCtx    context.Context
	workerCancel context.CancelFunc
	workStream   chan *values.CouchbaseCluster
	numWorkers   int
	streamSize   int
	trigger      chan struct{}

	// checkers is here so that during testing we can switch the checkers for test checkers.
	checkers map[string]values.CheckerFn

	progressMonitor *progress.Monitor
}

func newAPIMonitor(store storage.Store, streamSize int, workers int) *apiMonitor {
	return &apiMonitor{
		store:           store,
		streamSize:      streamSize,
		numWorkers:      workers,
		workerWg:        &sync.WaitGroup{},
		trigger:         make(chan struct{}, 10),
		checkers:        allCheckerFns,
		progressMonitor: progress.NewMonitor(),
	}
}

func (m *apiMonitor) start(APICheckerFrequency time.Duration) {
	// monitor already running
	if m.ctx != nil {
		return
	}

	zap.S().Infow("(Status Monitor) (API) Starting monitor", "frequency", APICheckerFrequency)
	m.workStream = make(chan *values.CouchbaseCluster, m.streamSize)
	m.ctx, m.cancel = context.WithCancel(context.Background())

	m.workerCtx, m.workerCancel = context.WithCancel(context.Background())
	for i := 0; i < m.numWorkers; i++ {
		m.workerWg.Add(1)
		go (&apiWorker{
			store:           m.store,
			progressMonitor: m.progressMonitor,
			checkers:        m.checkers,
			ctx:             m.workerCtx,
			stream:          m.workStream,
			wg:              m.workerWg,
		}).start()
	}

	m.wg.Add(1)
	m.trigger <- struct{}{}
	go m.periodicAPICheck(APICheckerFrequency)
}

func (m *apiMonitor) stop() {
	// not running
	if m.ctx != nil {
		return
	}

	zap.S().Info("(Status Monitor) (API) Stopping monitor")
	// stop work distribution
	m.cancel()
	m.wg.Wait()

	m.ctx, m.cancel = nil, nil

	// stop workers
	close(m.workStream)
	m.workerCancel()
	m.workerWg.Wait()

	m.workerCtx, m.workerCancel = nil, nil
}

func (m *apiMonitor) triggerCheck() error {
	if m.ctx == nil {
		return fmt.Errorf("api status monitor not running")
	}

	if len(m.trigger) >= 10 {
		return fmt.Errorf("api status monitor already waiting to run")
	}

	m.trigger <- struct{}{}
	return nil
}

func (m *apiMonitor) triggerFor(cluster *values.CouchbaseCluster) error {
	if m.ctx == nil {
		return fmt.Errorf("api status monitor not running")
	}

	m.workStream <- cluster
	return nil
}

func (m *apiMonitor) periodicAPICheck(frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer func() {
		m.wg.Done()
		ticker.Stop()
	}()

	for {
		select {
		case <-m.trigger:
			m.distributeWorkload()
		case <-ticker.C:
			m.distributeWorkload()
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *apiMonitor) distributeWorkload() {
	zap.S().Debugw("(Status Monitor) (API) API check tick")
	start := time.Now()
	clusters, err := m.store.GetClusters(true)
	if err != nil {
		zap.S().Errorw("(Status Monitor) (API) Could not get clusters", "err", err)
		return
	}

	m.progressMonitor.StartChecking(clusters)
	defer m.progressMonitor.FinishChecking()

	for _, c := range clusters {
		// short circuit
		select {
		case <-m.ctx.Done():
			return
		default:
			m.workStream <- c
		}
	}

	zap.S().Debugw("(Status Monitor) (API) API check tick", "elapsed", time.Since(start).String())
}
