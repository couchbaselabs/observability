package progress

import (
	"sync"
	"time"

	"github.com/couchbaselabs/cbmultimanager/values"
)

type Monitor struct {
	inProgress      bool
	lastRun         time.Time
	mapLock         sync.RWMutex
	clusterProgress values.ClusterProgressMap
}

func NewMonitor() *Monitor {
	return &Monitor{
		clusterProgress: make(values.ClusterProgressMap),
	}
}

func (m *Monitor) StartChecking(clusters []*values.CouchbaseCluster) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()

	m.inProgress = true
	m.lastRun = time.Now().UTC()
	m.clusterProgress = make(values.ClusterProgressMap)

	for _, c := range clusters {
		m.clusterProgress[c.UUID] = &values.ClusterProgress{Status: values.Waiting}
	}
}

func (m *Monitor) FinishChecking() {
	m.inProgress = false
}

func (m *Monitor) ClusterRunStart(uuid string, checkNum int) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()

	m.clusterProgress[uuid] = &values.ClusterProgress{
		Status:        values.InProgress,
		TotalCheckers: checkNum,
		Start:         now(),
	}
}

func (m *Monitor) ClusterRunEnd(uuid string) error {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()

	if _, ok := m.clusterProgress[uuid]; !ok {
		return &ClusterNotFoundError{uuid: uuid}
	}

	m.clusterProgress[uuid].End = now()
	m.clusterProgress[uuid].Status = values.Done
	return nil
}

func (m *Monitor) CheckerDone(uuid string, failed bool) error {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()

	if _, ok := m.clusterProgress[uuid]; !ok {
		return &ClusterNotFoundError{uuid: uuid}
	}

	if failed {
		m.clusterProgress[uuid].Failed++
		return nil
	}

	m.clusterProgress[uuid].Done++
	return nil
}

func (m *Monitor) GetProgressFor(uuid string) (*values.ClusterProgress, error) {
	m.mapLock.RLock()
	defer m.mapLock.RUnlock()

	progress, ok := m.clusterProgress[uuid]
	if !ok {
		return nil, &ClusterNotFoundError{uuid: uuid}
	}

	progressCopy := *progress
	return &progressCopy, nil
}

func now() *time.Time {
	now := time.Now().UTC()
	return &now
}
