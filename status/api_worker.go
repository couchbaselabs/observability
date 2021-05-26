package status

import (
	"context"
	"sync"
	"time"

	"github.com/couchbaselabs/cbmultimanager/statistics"
	"github.com/couchbaselabs/cbmultimanager/status/progress"
	"github.com/couchbaselabs/cbmultimanager/storage"
	"github.com/couchbaselabs/cbmultimanager/values"

	"go.uber.org/zap"
)

type apiWorker struct {
	store           storage.Store
	progressMonitor *progress.Monitor
	checkers        map[string]values.CheckerFn

	ctx    context.Context
	stream <-chan *values.CouchbaseCluster
	wg     *sync.WaitGroup
}

func (w *apiWorker) start() {
	defer w.wg.Done()

	for cluster := range w.stream {
		if w.ctx.Err() != nil {
			return
		}

		zap.S().Infow("(Status Monitor) (API) (Worker) Running checkers for cluster", "cluster", cluster.UUID)
		start := time.Now()

		// for each cluster in the stream run all the checkers
		w.runCheckers(cluster)

		zap.S().Debugw("(Status Monitor) (API) (Worker) All checks run", "elapsed", time.Since(start).String())
	}
}

func (w *apiWorker) runCheckers(cluster *values.CouchbaseCluster) {
	w.progressMonitor.ClusterRunStart(cluster.UUID, len(w.checkers))
	defer func() {
		if err := w.progressMonitor.ClusterRunEnd(cluster.UUID); err != nil {
			zap.S().Errorw("(Status Monitor) (API) (Worker) Closing cluster progress failed", "uuid", cluster.UUID,
				"err", err)
		}
	}()

	for name, checker := range w.checkers {
		if w.ctx.Err() != nil {
			return
		}

		zap.S().Debugw("(Status Monitor) (API) (Worker) Running checker", "fnName", name, "cluster", cluster.UUID)
		results, err := checker(cluster)

		if checkerErr := w.progressMonitor.CheckerDone(cluster.UUID, err != nil); err != nil {
			zap.S().Errorw("(Status Monitor) (API) (Worker) Could not update cluster progress", "uuid", cluster.UUID,
				"err", checkerErr)
		}

		// increments prometheus metrics
		statistics.CheckStatus(results)

		if err != nil {
			zap.S().Errorw("(Status Monitor) (API) (Worker) Could not run checker", "fnName", name, "cluster",
				cluster.UUID, "err", err)
			continue
		}

		// store the results in the store
		w.storeResults(cluster.UUID, name, results)
	}
}

func (w *apiWorker) storeResults(uuid, name string, results []*values.WrappedCheckerResult) {
	for _, res := range results {
		if res.Error != nil {
			zap.S().Errorw("(Status Monitor) (API) (Worker) Encountered error on checker function", "fnName",
				name, "err", res.Error, "cluster", uuid)
			continue
		}

		if err := w.store.SetCheckerResult(res); err != nil {
			zap.S().Errorw("(Status Monitor) (API) (Worker) Could not store checker result", "err", err,
				"fName", name, "checker", res.Result.Name, "cluster", uuid)
		}
	}
}
