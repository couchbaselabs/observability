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

package manager

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/couchbaselabs/observability/config-svc/pkg/metacfg"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var (
	activeClusterInfoStreamingConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "cmoscfg",
		Subsystem: "manager",
		Name:      "active_cluster_info_streaming_connections",
		Help:      "Number of currently active streaming connections for cluster information",
	})
	streamingClusterInfoErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "cmoscfg",
		Subsystem: "manager",
		Name:      "streaming_cluster_info_errors",
		Help:      "Number of errors when streaming cluster information",
	})
)

func init() {
	prometheus.MustRegister(streamingClusterInfoErrors, activeClusterInfoStreamingConnections)
}

type ClusterManager struct {
	logger *zap.SugaredLogger
	cfg    metacfg.ConfigManager
	// clusters holds a mapping of cluster UUID to cluster state.
	clusters       map[string]*clusterState
	clustersMux    sync.RWMutex
	pollingLoopCtx context.Context
	listeners      map[ClusterInfoListener]bool
	listenersMux   sync.RWMutex
	backoffPolicy  backoff.BackOff
}

var defaultBackoffPolicy = backoff.NewExponentialBackOff()

func NewClusterManager(baseLogger *zap.Logger, cfg metacfg.ConfigManager) (*ClusterManager, error) {
	cm := ClusterManager{
		logger:         baseLogger.Named("clusterManager").Sugar(),
		cfg:            cfg,
		clusters:       make(map[string]*clusterState),
		listeners:      make(map[ClusterInfoListener]bool),
		pollingLoopCtx: context.TODO(),
		backoffPolicy:  defaultBackoffPolicy,
	}
	return &cm, nil
}

func (m *ClusterManager) initializeClusters() {
	m.clustersMux.Lock()
	defer m.clustersMux.Unlock()
	cfg := m.cfg.Get()
	for _, cluster := range cfg.Clusters {
		m.initializeCluster(cluster)
	}
	m.logger.Debug("Cluster initialization complete.")
}

func (m *ClusterManager) initializeCluster(cluster metacfg.ClusterConfig) {
	var success bool
	for _, node := range cluster.Nodes.GetNodes() {
		var clusterPayload interface{}
		err := m.makeRequestToNode(node, cluster.CouchbaseConfig, "/pools/default/terseClusterInfo",
			&clusterPayload)
		if err != nil {
			m.logger.Warnw("Failed to get cluster info. Trying next node.",
				"cluster", cluster,
				"node", node, "err", err)
			continue
		}
		var uuid string
		switch data := clusterPayload.(type) {
		case map[string]interface{}:
			uuid = data["clusterUUID"].(string)
		default:
			m.logger.Warnw("Got unexpected clusterInfo. Trying next node.",
				"cluster", cluster,
				"node", node, "err", err, "info", data)
			continue
		}

		var nodesPayload poolsDefault
		err = m.makeRequestToNode(node, cluster.CouchbaseConfig, "/pools/default", &nodesPayload)
		if err != nil {
			m.logger.Warnw("Failed to get cluster nodes information. Trying next node.",
				"cluster", cluster,
				"node", node,
				"err", err)
			continue
		}
		m.logger.Debugw("Got cluster nodes", "uuid", uuid, "node", node, "cfg", nodesPayload)
		// applyClusterConfig will fill in the blanks
		m.clusters[uuid] = &clusterState{
			cfg:          &cluster,
			uuid:         uuid,
			currentNodes: make([]string, len(nodesPayload.Nodes)),
		}
		for i, node := range nodesPayload.Nodes {
			host, _, err := net.SplitHostPort(node.Hostname)
			if err != nil {
				m.logger.Errorw("Couchbase API gave us an invalid hostport!",
					"node", node,
					"err", err)
				continue
			}
			m.clusters[uuid].currentNodes[i] = host
		}
		success = true
		break
	}
	if !success {
		m.logger.Errorw("Failed to initialize cluster",
			"cluster", cluster)
		return
	}
}

func (m *ClusterManager) streamUpdatesFrom(cluster *clusterState, node string) error {
	// We'll first make a request to terseClusterInfo to check if this node is still in the cluster
	var terseClusterInfo interface{}
	err := m.makeRequestToNode(node, cluster.cfg.CouchbaseConfig, "/pools/default/terseClusterInfo",
		&terseClusterInfo)
	if err != nil {
		return fmt.Errorf("failed to fetch terseClusterInfo before stream from %v: %w", node, err)
	}
	if str, ok := terseClusterInfo.(string); ok {
		// if this is "unknown pool" that means the node's left the cluster
		// either way, it's not what we want
		return fmt.Errorf("got unexpected string from terseClusterInfo of node %v: %v", node, str)
	}
	updates, err := m.makeStreamingRequestToNode(m.pollingLoopCtx, node, cluster.cfg.CouchbaseConfig,
		"/poolsStreaming/default", new(poolsDefault))
	if err != nil {
		return fmt.Errorf("failed to initiate streaming connection to %v: %w", node, err)
	}
	m.logger.Debugw("Opened streaming pools connection", "node", node)
	activeClusterInfoStreamingConnections.Inc()
	defer activeClusterInfoStreamingConnections.Dec()
	for in := range updates {
		switch msg := in.(type) {
		case error:
			m.logger.Errorw("Received error from streaming connection",
				"node", node,
				"err", msg)
			return msg
		case *poolsDefault:
			m.clustersMux.Lock()
			val := clusterState{
				uuid:         cluster.uuid,
				currentNodes: make([]string, len(msg.Nodes)),
				cfg:          cluster.cfg,
			}
			for i, node := range msg.Nodes {
				host, _, err := net.SplitHostPort(node.Hostname)
				if err != nil {
					m.logger.Errorw("Couchbase API gave us an invalid hostport!",
						"node", node,
						"err", err)
					continue
				}
				val.currentNodes[i] = host
			}
			m.clusters[cluster.uuid] = &val
			m.clustersMux.Unlock()
			m.notifyClusterStateChange(val)
		default:
			m.logger.Warnw("Received unknown message type",
				"node", node,
				"msg", msg)
		}
	}
	m.logger.Infow("Streaming connection closed", "node", node)
	return nil
}

func (m *ClusterManager) StartUpdating() {
	m.initializeClusters()
	m.logger.Debug("Starting update loop")
	// Make a copy of the clusters map just in case it gets updated
	m.clustersMux.RLock()
	clusters := m.clusters
	m.clustersMux.RUnlock()
	for uuid := range clusters {
		go func(uuid string) {
			backoffPolicy := m.backoffPolicy
			for {
				err := backoff.RetryNotify(func() error {
					m.clustersMux.RLock()
					cluster := m.clusters[uuid]
					m.clustersMux.RUnlock()
					if len(cluster.currentNodes) == 0 {
						return fmt.Errorf("cluster has no known nodes")
					}
					return m.streamUpdatesFrom(cluster, cluster.currentNodes[0])
				}, backoffPolicy, func(err error, retryAfter time.Duration) {
					if err != nil {
						streamingClusterInfoErrors.Inc()
					}
					m.logger.Warnw("Streaming failed, waiting before retrying",
						"clusterUUID", uuid,
						"err", err,
						"retryAfter", retryAfter)
				})
				if err != nil {
					m.logger.Warnw("Failed to execute backoff",
						"clusterUUID", uuid,
						"err", err)
				}
			}
		}(uuid)
	}
}

func (m *ClusterManager) GetClusters() ([]ClusterInfo, error) {
	m.clustersMux.RLock()
	defer m.clustersMux.RUnlock()
	clusters := m.clusters
	result := make([]ClusterInfo, len(clusters))
	i := 0
	for _, cluster := range clusters {
		val := ClusterInfo{
			Nodes:           cluster.currentNodes,
			Metadata:        cluster.cfg.Metadata,
			CouchbaseConfig: cluster.cfg.CouchbaseConfig,
			MetricsConfig:   cluster.cfg.MetricsConfig,
		}
		result[i] = val
		i++
	}
	return result, nil
}

func (m *ClusterManager) notifyClusterStateChange(val clusterState) {
	m.logger.Debugw("Notifying about cluster state change", "clusterUUID", val.uuid, "nodes", val.currentNodes)
	m.listenersMux.RLock()
	defer m.listenersMux.RUnlock()
	for listener := range m.listeners {
		go func(l ClusterInfoListener, cs clusterState) {
			l <- ClusterInfo{
				UUID:            cs.uuid,
				Nodes:           cs.currentNodes,
				Metadata:        cs.cfg.Metadata,
				CouchbaseConfig: cs.cfg.CouchbaseConfig,
				MetricsConfig:   cs.cfg.MetricsConfig,
			}
		}(listener, val)
	}
}

func (m *ClusterManager) Subscribe(listener ClusterInfoListener) {
	m.listenersMux.Lock()
	defer m.listenersMux.Unlock()
	m.listeners[listener] = true
}

func (m *ClusterManager) Unsubscribe(listener ClusterInfoListener) {
	m.listenersMux.Lock()
	defer m.listenersMux.Unlock()
	delete(m.listeners, listener)
}
