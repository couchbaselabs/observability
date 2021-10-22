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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbaselabs/observability/config-svc/pkg/metacfg"
	"go.uber.org/zap"
)

type ClusterManager struct {
	logger *zap.SugaredLogger
	cfg    metacfg.ConfigManager
	// clusters holds a mapping of cluster UUID to cluster state.
	clusters       map[string]*clusterState
	clustersMux    sync.RWMutex
	pollingLoopCtx context.Context
}

func NewClusterManager(baseLogger *zap.Logger, cfg metacfg.ConfigManager) (*ClusterManager, error) {
	cm := ClusterManager{
		logger:         baseLogger.Named("clusterManager").Sugar(),
		cfg:            cfg,
		clusters:       make(map[string]*clusterState),
		pollingLoopCtx: context.TODO(),
	}
	return &cm, nil
}

type pools struct {
	UUID string `json:"uuid"`
}

func (m *ClusterManager) makeRequestToNode(node string, cfg metacfg.CouchbaseConfig, endpoint string,
	out interface{}) error {
	req, err := http.NewRequestWithContext(m.pollingLoopCtx, http.MethodGet,
		fmt.Sprintf("https://%s:%d%s",
			node,
			cfg.ManagementPort,
			endpoint), nil)
	if err != nil {
		return fmt.Errorf("failed to prepare HTTP request: %w", err)
	}
	req.SetBasicAuth(cfg.Username, cfg.Password)
	client := http.DefaultClient
	if cfg.IgnoreCertificateErrors {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read HTTP response body: %w", err)
	}
	err = json.Unmarshal(body, out)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return nil
}

func (m *ClusterManager) applyClusterConfig(uuid string, payload *cbrest.ClusterConfig) error {
	if m.clusters[uuid].configRevision > payload.Revision {
		return fmt.Errorf("received config too old; latest revision %d, given %d",
			m.clusters[uuid].configRevision,
			payload.Revision)
	}
	clusterNodes := make([]string, len(payload.Nodes))
	for i, newNode := range payload.Nodes {
		clusterNodes[i] = newNode.Hostname
	}
	m.clusters[uuid].configRevision = payload.Revision
	m.clusters[uuid].currentNodes = clusterNodes
	return nil
}

func (m *ClusterManager) initializeClusters() {
	m.clustersMux.Lock()
	defer m.clustersMux.Unlock()
	cfg := m.cfg.Get()
	for _, cluster := range cfg.Clusters {
		var success bool
		for _, node := range cluster.Nodes.GetNodes() {
			var clusterPayload pools
			err := m.makeRequestToNode(node, cluster.CouchbaseConfig, "/pools", &clusterPayload)
			if err != nil {
				m.logger.Warnw("Failed to get cluster config. Trying next node.",
					"cluster", cluster,
					"node", node, "err", err)
				continue
			}
			uuid := clusterPayload.UUID

			var nodesPayload cbrest.ClusterConfig
			err = m.makeRequestToNode(node, cluster.CouchbaseConfig, "/pools/default/nodeServices", &nodesPayload)
			if err != nil {
				m.logger.Warnw("Failed to get cluster config. Trying next node.",
					"cluster", cluster,
					"node", node,
					"err", err)
				continue
			}
			m.logger.Debugw("Got cluster config", "uuid", uuid, "node", node, "cfg", nodesPayload)
			// applyClusterConfig will fill in the blanks
			m.clusters[uuid] = &clusterState{
				cfg:            &cluster,
				configRevision: -1,
			}
			if err := m.applyClusterConfig(uuid, &nodesPayload); err != nil {
				m.logger.Warnw("Failed to apply cluster config. Trying next node.",
					"cluster", cluster,
					"cfg", nodesPayload,
					"err", err)
			}
			success = true
			break
		}
		if !success {
			m.logger.Errorw("Failed to initialize cluster",
				"cluster", cluster)
		}
	}
	m.logger.Debug("Cluster initialization complete.")
}

func (m *ClusterManager) updateClusters() {
	m.clustersMux.Lock()
	defer m.clustersMux.Unlock()
	for uuid, cluster := range m.clusters {
		var success bool
		for _, node := range cluster.currentNodes {
			var payload cbrest.ClusterConfig
			err := m.makeRequestToNode(node, cluster.cfg.CouchbaseConfig, "/pools/default/nodeServices", &payload)
			if err != nil {
				m.logger.Warnw("Failed to get cluster config. Trying next node.",
					"uuid", uuid,
					"node", node, "err", err)
				continue
			}
			m.logger.Debugw("Got cluster config", "uuid", uuid, "node", node, "cfg", payload)
			if err := m.applyClusterConfig(uuid, &payload); err != nil {
				m.logger.Warnw("Failed to apply cluster config. Trying next node.",
					"uuid", uuid,
					"cfg", payload,
					"err", err)
			}
			success = true
			break
		}
		if !success {
			m.logger.Warnw("Exhausted all currently nodes for cluster. Falling back to seed nodes.",
				"uuid", uuid)
			for _, node := range cluster.cfg.Nodes.GetNodes() {
				var payload cbrest.ClusterConfig
				err := m.makeRequestToNode(node, cluster.cfg.CouchbaseConfig, "/pools/default/nodeServices", &payload)
				if err != nil {
					m.logger.Warnw("Failed to get cluster config. Trying next node.",
						"uuid", uuid,
						"node", node, "err", err)
					continue
				}
				m.logger.Debugw("Got cluster config", "uuid", uuid, "node", node, "cfg", payload)
				if err := m.applyClusterConfig(uuid, &payload); err != nil {
					m.logger.Warnw("Failed to apply cluster config. Trying next node.",
						"uuid", uuid,
						"cfg", payload,
						"err", err)
				}
				success = true
				break
			}
			if !success {
				m.logger.Errorw("Exhausted all known nodes for cluster.",
					"uuid", uuid)
			}
		}
	}
}

func (m *ClusterManager) UpdateLoop() {
	m.initializeClusters()
	m.logger.Debug("Starting update loop")
	for {
		m.updateClusters()
		cfg := m.cfg.Get()
		m.logger.Debugw("Update loop run complete", "updateInterval", cfg.ClusterUpdateInterval)
		select {
		case <-m.pollingLoopCtx.Done():
			m.logger.Warnw("Cancelling update loop", "err", m.pollingLoopCtx.Err())
		case <-time.After(cfg.ClusterUpdateInterval):
		}
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
