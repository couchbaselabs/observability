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

package api

import (
	"fmt"
	"net/http"

	"gopkg.in/yaml.v3"

	v1 "github.com/couchbaselabs/observability/config-svc/pkg/api/v1"
	"github.com/labstack/echo/v4"
	"gopkg.in/guregu/null.v4"
)

func (s *Server) GetConfig(ctx echo.Context) error {
	cfg := s.cfg.Get()
	yamlValue, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return ctx.Blob(http.StatusOK, "text/yaml", yamlValue)
}

func (s *Server) GetClusters(ctx echo.Context, params v1.GetClustersParams) error {
	val, err := s.clusters.GetClusters()
	if err != nil {
		return err
	}
	result := make([]v1.CouchbaseCluster, len(val))
	for i, cluster := range val {
		result[i] = v1.CouchbaseCluster{
			Nodes:    cluster.Nodes,
			Metadata: v1.CouchbaseCluster_Metadata{AdditionalProperties: cluster.Metadata},
			CouchbaseConfig: v1.CouchbaseServerConfig{
				ManagementPort: float32(cluster.CouchbaseConfig.ManagementPort),
			},
		}
		if null.BoolFromPtr(params.IncludeSensitiveInfo).ValueOrZero() {
			result[i].CouchbaseConfig.Username = &cluster.CouchbaseConfig.Username
			result[i].CouchbaseConfig.Password = &cluster.CouchbaseConfig.Password
		}
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) GetPrometheusTargets(ctx echo.Context) error {
	val, err := s.clusters.GetClusters()
	if err != nil {
		return err
	}
	result := make([]v1.PrometheusScrapeConfig, len(val))
	for i, cluster := range val {
		result[i] = v1.PrometheusScrapeConfig{
			Labels: v1.PrometheusScrapeConfig_Labels{
				AdditionalProperties: cluster.Metadata,
			},
			Targets: make([]string, len(cluster.Nodes)),
		}
		for j, node := range cluster.Nodes {
			result[i].Targets[j] = fmt.Sprintf("%s:%d", node, cluster.MetricsConfig.ExporterPort)
		}
	}
	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) GetOpenapiJson(ctx echo.Context) error { //nolint:revive
	swagger, err := v1.GetSwagger()
	if err != nil {
		return err
	}
	return ctx.JSONPretty(http.StatusOK, swagger, "\t")
}
