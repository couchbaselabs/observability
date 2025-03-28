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
 "strings"

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
	mgmtPort := 8091
	if data.CouchbaseConfig.UseTLS != nil && *data.CouchbaseConfig.UseTLS {
		useTLS = true
		scheme = "https"
	}

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
	var cfg  prometheus.Configuration

	var username= data.CouchbaseConfig.Username
	var password =data.CouchbaseConfig.Password

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

	if err := yaml.Unmarshal(existingConfig, &cfg); err != nil {
		return fmt.Errorf("failed to parse Prometheus config: %w", err)
	}
	cbScrapeConfig, err := createScrapeConfigForCluster(cluster, mgmtPort,useTLS, username, password, data.MetricsConfig)
	if err != nil {
		return err
	}

	// Job name needs to be unique
	cbScrapeConfig.JobName = fmt.Sprintf("couchbase-server-managed-%d", len(cfg.ScrapeConfigs)+1)

	// Sync Gateway metrics path is metrics
	cbScrapeConfig.MetricsPath = "/metrics"


	addScrapeConfigIfUnique(&cfg, cbScrapeConfig, ScrapeConfigDeduplicationCriteria{
		CheckTargetsAndPath: true,
	})


	xdcrScrapeConfig := createXDCREndpointScrapeConfig(cluster,mgmtPort, username, password,useTLS)


	addScrapeConfigIfUnique(&cfg, xdcrScrapeConfig, ScrapeConfigDeduplicationCriteria{
		CheckJobName: true,
	})



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

type ScrapeConfigDeduplicationCriteria struct {
	CheckJobName        bool
	CheckTargetsAndPath bool
}

func addScrapeConfigIfUnique(cfg *prometheus.Configuration, newCfg *prometheus.ScrapeConfig, criteria ScrapeConfigDeduplicationCriteria) {
	for _, existing := range cfg.ScrapeConfigs {
		// Check by job name
		if criteria.CheckJobName && existing.JobName == newCfg.JobName {
			return
		}

		// Check by target + metrics path
		if criteria.CheckTargetsAndPath && existing.MetricsPath == newCfg.MetricsPath {
			for _, existingStatic := range existing.StaticConfigs {
				for _, existingTarget := range existingStatic.Targets {
					for _, newStatic := range newCfg.StaticConfigs {
						for _, newTarget := range newStatic.Targets {
							if existingTarget == newTarget {
								return // Duplicate found
							}
						}
					}
				}
			}
		}
	}
	// If no duplicates match the chosen criteria, append
	cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, newCfg)
}


func (s *Server) PostSgwAdd(ctx echo.Context) error {
	var data v1.PostSgwAddJSONRequestBody
	// need to write new scrape config for SGW
	if err := ctx.Bind(&data); err != nil {
		return err
	}

	metricsPort := 4986

	scrapeConfig := createScrapeConfigForSGW(
		data.SgwConfig.Username,
		data.SgwConfig.Password,
		data.Hostname,
		metricsPort,
	)

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
	scrapeConfig.JobName = fmt.Sprintf("sync-gateway-managed-%d", len(cfg.ScrapeConfigs)+1)

	// Sync Gateway metrics path is _metrics
	scrapeConfig.MetricsPath = "/_metrics"

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


func createXDCREndpointScrapeConfig(cluster *couchbase.PoolsDefault, managementport int , username, password string, useTLS bool) *prometheus.ScrapeConfig {
	var schema=  map[bool]string{true: "https", false: "http"}[useTLS]
	var hostname=  strings.Split(cluster.Nodes[0].Hostname, ":")[0]
	var result= prometheus.ScrapeConfig{
		JobName:     fmt.Sprintf("%s-http", cluster.ClusterName),
		MetricsPath: "/probe",
		HTTPClientConfig: prometheus.HTTPClientConfig{
			//FOR PROM EXPORTER IS ALWAYS HTTP
			Schema: "http",
			BasicAuth: prometheus.BasicAuthConfig{
				Username: username,
				Password: password,
			},
		},
		StaticConfigs: []prometheus.StaticConfig{
			{

				Targets: []string{"localhost:7979"}, // JSON exporter target
			},
		},
		// Params are how json_exporter gets the actual target URL
		Params: map[string][]string{
			"module": {"pools"},
			//THE TARGET TAKES THE USETLS ARGUMENT
			"target": {fmt.Sprintf("%s://%s:%d/pools/default/tasks",schema,hostname,managementport)},
		},
		RelabelConfigs: []prometheus.RelabelConfig{
			{
				TargetLabel: "cluster",
				Replacement: cluster.ClusterName,
			},
		},
	}
	return &result
}



func createScrapeConfigForCluster(cluster *couchbase.PoolsDefault, managementport int , useTLS bool, username, password string,
	metricsConfig MetricsConfig) (*prometheus.ScrapeConfig, error) {
	var schema=  map[bool]string{true: "https", false: "http"}[useTLS]


	staticConfig := prometheus.StaticConfig{
		Targets: make([]string, len(cluster.Nodes)),

		Labels: map[string]string{
			"cluster_name": cluster.ClusterName,
		},
	}


	for i, node := range cluster.Nodes {
		var hostname= strings.Split(node.Hostname, ":")[0]

		staticConfig.Targets[i] = fmt.Sprintf("%s:%d", hostname, managementport)


	}

	scrapeConfig := prometheus.ScrapeConfig{
		StaticConfigs: []prometheus.StaticConfig{staticConfig},
	}

	scrapeConfig.HTTPClientConfig = prometheus.HTTPClientConfig{
			Schema: schema,
			BasicAuth: prometheus.BasicAuthConfig{
				Username: username,
				Password: password,
			},
	}


	return &scrapeConfig, nil
}

func createScrapeConfigForSGW(username, password string, hostname string,
	metricsPort int) *prometheus.ScrapeConfig {
	staticConfig := prometheus.StaticConfig{
		Targets: make([]string, 1),
	}

	staticConfig.Targets[0] = fmt.Sprintf("%s:%d", hostname, metricsPort)

	scrapeConfig := prometheus.ScrapeConfig{
		StaticConfigs: []prometheus.StaticConfig{staticConfig},
	}
	scrapeConfig.HTTPClientConfig = prometheus.HTTPClientConfig{

		BasicAuth: prometheus.BasicAuthConfig{
			Username: username,
			Password: password,
		},
	}

	return &scrapeConfig
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
