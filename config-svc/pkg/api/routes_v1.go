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
	"os/exec"

	"github.com/couchbase/tools-common/cbvalue"
	"github.com/couchbaselabs/observability/config-svc/pkg/couchbase"
	"github.com/couchbaselabs/observability/config-svc/pkg/prometheus"
	"gopkg.in/yaml.v3"

	v1 "github.com/couchbaselabs/observability/config-svc/pkg/api/v1"
	"github.com/labstack/echo/v4"
)

const (
	defaultPrometheusConfigPath = "/etc/prometheus/prometheus.yml"
	collectInfoPath             = "/collect-information.sh"
)

func (s *Server) PostClustersAdd(ctx echo.Context) error {
	var data v1.PostClustersAddJSONRequestBody
	if err := ctx.Bind(&data); err != nil {
		return err
	}

	scheme := "http"
	useTLS := false
	if data.CouchbaseConfig.UseTLS != nil && *data.CouchbaseConfig.UseTLS {
		useTLS = true
		scheme = "https"
	}
	mgmtPort := 8091
	if data.CouchbaseConfig.ManagementPort != nil {
		mgmtPort = int(*data.CouchbaseConfig.ManagementPort)
	}
	cluster, err := couchbase.FetchCouchbaseClusterInfo(
		scheme,
		data.Hostname,
		mgmtPort,
		data.CouchbaseConfig.Username,
		data.CouchbaseConfig.Password,
	)
	if err != nil {
		return fmt.Errorf("unable to get cluster info: %w", err)
	}

	scrapeConfig, err := createScrapeConfigForCluster(
		cluster,
		useTLS,
		data.CouchbaseConfig.Username,
		data.CouchbaseConfig.Password,
		data.MetricsConfig,
	)
	if err != nil {
		return fmt.Errorf("could not create scrape config: %w", err)
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

	// Job name needs to be unique
	scrapeConfig.JobName = fmt.Sprintf("couchbase-server-managed-%d", len(cfg.ScrapeConfigs)+1)

	cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scrapeConfig)

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

func (s *Server) PostCollectInformation(ctx echo.Context) error {
	cmd := exec.Command(collectInfoPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}
	return ctx.Stream(http.StatusOK, "text/plain", stdout)
}

type MetricsConfig *struct {
	MetricsPort *float32 `json:"metricsPort,omitempty"`
}

func createScrapeConfigForCluster(cluster *couchbase.PoolsDefault, useTLS bool, username, password string,
	metricsConfig MetricsConfig) (*prometheus.ScrapeConfig, error) {
	staticConfig := prometheus.StaticConfig{
		Targets: make([]string, len(cluster.Nodes)),
		Labels: map[string]string{
			"cluster": cluster.ClusterName,
		},
	}

	var anyNodeCB7 bool

	for i, node := range cluster.Nodes {
		hostname, mgmtPort, err := node.ResolveHostPort(useTLS)
		if err != nil {
			return nil, err
		}
		if node.Version.AtLeast(cbvalue.Version7_0_0) {
			staticConfig.Targets[i] = fmt.Sprintf("%s:%d", hostname, mgmtPort)
			anyNodeCB7 = true
		} else if metricsConfig != nil && metricsConfig.MetricsPort != nil {
			staticConfig.Targets[i] = fmt.Sprintf("%s:%.0f", hostname, *metricsConfig.MetricsPort)
		} else {
			staticConfig.Targets[i] = fmt.Sprintf("%s:%d", hostname, 9091)
		}
	}

	scrapeConfig := prometheus.ScrapeConfig{
		StaticConfigs: []prometheus.StaticConfig{staticConfig},
	}
	if anyNodeCB7 {
		scrapeConfig.HTTPClientConfig = prometheus.HTTPClientConfig{
			BasicAuth: prometheus.BasicAuthConfig{
				Username: username,
				Password: password,
			},
		}
	}

	return &scrapeConfig, nil
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
