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
	"encoding/json"
	"fmt"
	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbaselabs/observability/config-svc/pkg/metacfg"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

type ClusterManager struct {
	logger *zap.SugaredLogger
	cfg    metacfg.ConfigManager
	// clusters holds a mapping of cluster UUID to cluster state.
	clusters map[string]*Cluster
	ctx      context.Context
}

func NewClusterManager(baseLogger *zap.Logger, cfg metacfg.ConfigManager) (*ClusterManager, error) {
	cm := ClusterManager{
		logger:   baseLogger.Named("clusterManager").Sugar(),
		cfg:      cfg,
		clusters: make(map[string]*Cluster),
		ctx:      context.TODO(),
	}
	return &cm, nil
}

func (m *ClusterManager) updateClusters() {
	for uuid, cluster := range m.clusters {
		var clusterNodes []string
		var success bool
		for _, node := range cluster.currentNodes {
			req, err := http.NewRequestWithContext(m.ctx, http.MethodGet,
				fmt.Sprintf("https://%s:%d/pools/default/nodeServices",
					node,
					cluster.cfg.CouchbaseConfig.ManagementPort), nil)
			if err != nil {
				m.logger.Errorw("Failed to update cluster: could not prepare HTTP request, trying next node.",
					"uuid", uuid,
					"node", node,
					"err", err)
				continue
			}
			req.SetBasicAuth(cluster.cfg.CouchbaseConfig.Username, cluster.cfg.CouchbaseConfig.Password)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				m.logger.Errorw("Failed to update cluster: could not execute HTTP request, trying next node.",
					"uuid", uuid,
					"node", node,
					"err", err)
				continue
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				m.logger.Errorw("Failed to update cluster: could not read HTTP body, trying next node.",
					"uuid", uuid,
					"node", node,
					"err", err)
				continue
			}
			var payload cbrest.ClusterConfig
			err = json.Unmarshal(body, &payload)
			if err != nil {
				m.logger.Errorw("Failed to update cluster: could not unmarshal HTTP body, trying next node.",
					"uuid", uuid,
					"node", node,
					"err", err)
				continue
			}
			m.logger.Debugw("Got cluster config", "uuid", uuid, "node", node, "cfg", payload)
			clusterNodes = make([]string, len(payload.Nodes))
			for i, newNode := range payload.Nodes {
				clusterNodes[i] = newNode.Hostname
			}
			m.clusters[uuid].configRevision = payload.Revision
			m.clusters[uuid].currentNodes = clusterNodes
			break
		}
		if !success {
			m.logger.Errorw("Exhausted all known nodes for cluster.", "uuid", uuid)
		}
	}
}

func (m *ClusterManager) UpdateLoop() {
	m.logger.Debug("Starting update loop")
	for {
		m.updateClusters()
		cfg := m.cfg.Get()
		m.logger.Debugw("Update loop run complete", "updateInterval", cfg.ClusterUpdateInterval)
		select {
		case <-m.ctx.Done():
			m.logger.Warnw("Cancelling update loop", "err", m.ctx.Err())
		case <-time.After(cfg.ClusterUpdateInterval):
		}
	}
}
