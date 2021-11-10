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
	"io"
	"net/http"
	"os"

	"github.com/couchbase/tools-common/cbvalue"
	"github.com/couchbaselabs/observability/config-svc/pkg/couchbase"
	"github.com/couchbaselabs/observability/config-svc/pkg/prometheus"
	"gopkg.in/yaml.v3"

	v1 "github.com/couchbaselabs/observability/config-svc/pkg/api/v1"
	"github.com/labstack/echo/v4"
)

const defaultPrometheusConfigPath = "/etc/prometheus/prometheus.yml"

func (s *Server) PostClustersAdd(ctx echo.Context) error {
	var data v1.PostClustersAddJSONRequestBody
	if err := ctx.Bind(&data); err != nil {
		return err
	}

	secure := false
	if data.CouchbaseConfig.Secure != nil && *data.CouchbaseConfig.Secure {
		secure = true
	}
	mgmtPort := 8091
	if data.CouchbaseConfig.ManagementPort != nil {
		mgmtPort = int(*data.CouchbaseConfig.ManagementPort)
	}
	cluster, err := couchbase.FetchCouchbaseClusterInfo(
		data.Hostname,
		mgmtPort,
		secure,
		data.CouchbaseConfig.Username,
		data.CouchbaseConfig.Password,
	)
	if err != nil {
		return err
	}

	staticConfig := prometheus.StaticConfig{
		Targets: make([]string, len(cluster.Nodes)),
		Labels: map[string]string{
			"cluster": cluster.ClusterName,
		},
	}

	var anyNodeCB7 bool

	for i, node := range cluster.Nodes {
		hostname, mgmtPort, err := node.ResolveHostPort(secure)
		if err != nil {
			return err
		}
		if node.Version.AtLeast(cbvalue.Version7_0_0) {
			staticConfig.Targets[i] = fmt.Sprintf("%s:%d", hostname, mgmtPort)
			anyNodeCB7 = true
		} else {
			// TODO: allow customising this
			staticConfig.Targets[i] = fmt.Sprintf("%s:%d", hostname, 9091)
		}
	}

	cfgPath := os.Getenv("PROMETHEUS_CONFIG_FILE")
	if cfgPath == "" {
		cfgPath = defaultPrometheusConfigPath
	}
	cfgFile, err := os.OpenFile(cfgPath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open Prometheus config: %w", err)
	}
	defer cfgFile.Close()
	existingConfig, err := io.ReadAll(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to read Prometheus config: %w", err)
	}
	var cfg prometheus.Configuration
	if err := yaml.Unmarshal(existingConfig, &cfg); err != nil {
		return fmt.Errorf("failed to parse Prometheus config: %w", err)
	}

	scrapeConfig := prometheus.ScrapeConfig{
		// Job name needs to be unique
		JobName:       fmt.Sprintf("couchbase-server-managed-%d", len(cfg.ScrapeConfigs)+1),
		StaticConfigs: []prometheus.StaticConfig{staticConfig},
	}
	if anyNodeCB7 {
		scrapeConfig.HTTPClientConfig = prometheus.HTTPClientConfig{
			BasicAuth: prometheus.BasicAuthConfig{
				Username: data.CouchbaseConfig.Username,
				Password: data.CouchbaseConfig.Password,
			},
		}
	}

	cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, &scrapeConfig)

	configYaml, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal Prometheus config: %w", err)
	}

	err = overwriteFileContents(cfgFile, configYaml)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

func overwriteFileContents(file *os.File, contents []byte) error {
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate Prometheus config: err")
	}

	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek Prometheus config: %w", err)
	}

	if _, err := file.Write(contents); err != nil {
		return fmt.Errorf("failed to write Prometheus config: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close Prometheus config: %w", err)
	}
	return nil
}

func (s *Server) GetOpenapiJson(ctx echo.Context) error { //nolint:revive
	swagger, err := v1.GetSwagger()
	if err != nil {
		return err
	}
	return ctx.JSONPretty(http.StatusOK, swagger, "\t")
}
