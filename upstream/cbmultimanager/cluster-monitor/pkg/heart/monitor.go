// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package heart

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"go.uber.org/zap"
)

// Monitor is the structure that will be in charge of periodically checking on the registered clusters.
type Monitor struct {
	store storage.Store

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	workStream chan *values.CouchbaseCluster
	numWorkers int
	workerWg   sync.WaitGroup
}

func NewMonitor(store storage.Store, workers int) *Monitor {
	return &Monitor{store: store, numWorkers: workers}
}

func (m *Monitor) Start(heartBeatFrequency time.Duration) {
	// monitor already running
	if m.ctx != nil {
		return
	}

	zap.S().Infow("(Heart Monitor) Starting monitor", "frequency", heartBeatFrequency)
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.wg.Add(1)
	go m.heartBeat(heartBeatFrequency)
}

func (m *Monitor) Stop() {
	// not running
	if m.ctx == nil {
		return
	}

	zap.S().Info("(Heart Monitor) Stopping monitor")
	m.cancel()
	m.wg.Wait()
	m.ctx, m.cancel = nil, nil
}

func (m *Monitor) heartBeat(heartBeatFrequency time.Duration) {
	ticker := time.NewTicker(heartBeatFrequency)
	defer func() {
		m.wg.Done()
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			if err := m.doClustersHeartBeat(); err != nil {
				zap.S().Warnw("(Heart Monitor) There was an issue performing clusters heartbeat", "err", err.Error())
			}
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Monitor) doClustersHeartBeat() error {
	zap.S().Infow("(Heart Monitor) Starting heartbeat")
	start := time.Now()
	clusters, err := m.store.GetClusters(true, false)
	if err != nil {
		return fmt.Errorf("could not get clusters to perform heartbeat: %w", err)
	}

	m.workStream = make(chan *values.CouchbaseCluster)
	// start the workers
	for i := 0; i < m.numWorkers; i++ {
		m.workerWg.Add(1)
		go m.heartBeatWorkerFn()
	}

	// send the data
	for _, cluster := range clusters {
		m.workStream <- cluster
	}

	close(m.workStream)

	// to avoid starting the next heartbeat before finishing this one we wait until all the workers are done
	m.workerWg.Wait()

	zap.S().Debugw("(Heart Monitor) heartbeat finished", "elapsed", time.Since(start).String(), "#clusters",
		len(clusters))
	return nil
}

func (m *Monitor) heartBeatWorkerFn() {
	defer m.workerWg.Done()

	for cluster := range m.workStream {
		if err := m.HeartBeatCluster(cluster); err != nil {
			zap.S().Errorw("(Heart Monitor) Could not update cluster state", "uuid", cluster.UUID, "err", err)
		}
	}
}

func (m *Monitor) HeartBeatCluster(cluster *values.CouchbaseCluster) error {
	zap.S().Debugw("(Heart Monitor) Heart beat for cluster", "uuid", cluster.UUID, "hosts",
		cluster.NodesSummary.GetHosts())
	client, err := couchbase.NewClient(cluster.NodesSummary.GetHosts(), cluster.User, cluster.Password,
		cluster.GetTLSConfig(), false)
	// in failure cases update cluster entry to reflect issue
	if err != nil {
		zap.S().Warnw("(Heart Monitor) Cluster heartbeat failed", "uuid", cluster.UUID, "err", err)
		issue := values.NoConnectionHeartIssue
		var authError couchbase.AuthError
		if errors.As(err, &authError) {
			issue = values.BadAuthHeartIssue
		}

		return m.store.UpdateCluster(&values.CouchbaseCluster{
			UUID:           cluster.UUID,
			HeartBeatIssue: issue,
		})
	}

	// check that the uuid has not changed
	if client.ClusterInfo.ClusterUUID != cluster.UUID {
		zap.S().Warnw("(Heart Monitor) Cluster UUID changed", "old", cluster.UUID, "new",
			client.ClusterInfo.ClusterUUID)
		return m.store.UpdateCluster(&values.CouchbaseCluster{
			UUID:           cluster.UUID,
			HeartBeatIssue: values.UUIDMismatchHeartIssue,
			Enterprise:     client.ClusterInfo.Enterprise,
		})
	}

	var remoteClusters values.RemoteClusters
	remoteClusters, err = client.GetRemoteClusters(client.ClusterInfo.ClusterName, client.ClusterInfo.ClusterUUID)
	if err != nil {
		zap.S().Errorw("(Heart Monitor) Could not update remote clusters", "cluster", cluster.UUID, "err", err)
	}

	// get up to date buckets information. If we fail to get the buckets we still update all the other stuff
	var buckets values.BucketsSummary
	buckets, err = client.GetBucketsSummary()
	if err != nil {
		zap.S().Errorw("(Heart Monitor) Could not update buckets summary", "cluster", cluster.UUID, "err", err)
	}

	// otherwise the heartbeat is OK so we just update the hosts and cluster name
	return m.store.UpdateCluster(&values.CouchbaseCluster{
		UUID:           cluster.UUID,
		Enterprise:     client.ClusterInfo.Enterprise,
		NodesSummary:   client.ClusterInfo.NodesSummary,
		Name:           client.ClusterInfo.ClusterName,
		ClusterInfo:    client.ClusterInfo.ClusterInfo,
		HeartBeatIssue: values.NoHeartIssue,
		BucketsSummary: buckets,
		RemoteClusters: remoteClusters,
	})
}
