package status

import (
	"time"

	"github.com/couchbaselabs/cbmultimanager/storage"
	"github.com/couchbaselabs/cbmultimanager/values"

	"go.uber.org/zap"
)

// Monitor is in charge of running the different status checks on the clusters. This will run various types of
// sub-monitors
type Monitor struct {
	apiMonitor *apiMonitor
}

func NewMonitor(store storage.Store, workers int) MonitorInterface {
	return MonitorInterface(&Monitor{
		apiMonitor: newAPIMonitor(store, workers*4, workers),
	})
}

func (m *Monitor) Start(APICheckerFrequency time.Duration) {
	zap.S().Infow("(Status Monitor) Starting monitor")
	m.apiMonitor.start(APICheckerFrequency)
}

func (m *Monitor) Stop() {
	zap.S().Info("(Status Monitor) Stopping monitor")
	m.apiMonitor.stop()
}

func (m *Monitor) TriggerAPICheck(cluster *values.CouchbaseCluster) error {
	if cluster == nil {
		return m.apiMonitor.triggerCheck()
	}

	return m.apiMonitor.triggerFor(cluster)
}

func (m *Monitor) GetProgressFor(uuid string) (*values.ClusterProgress, error) {
	return m.apiMonitor.progressMonitor.GetProgressFor(uuid)
}
