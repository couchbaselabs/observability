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

package metacfg

import (
	"fmt"
	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
	"time"
)

type Config struct {
	Clusters              []ClusterConfig        `yaml:"clusters" default:"[]" validate:"dive,required"`
	ClusterUpdateInterval time.Duration          `yaml:"cluster_update_interval" default:"15s"`
	PrometheusHosts       []string               `yaml:"prometheus_hosts" default:"[]" validate:"dive,url"`
	ClusterMonitorHosts   []ClusterMonitorConfig `yaml:"cluster_monitor_hosts" default:"[]" validate:"dive,required"`
	Immutable             bool                   `yaml:"immutable"`
	Server                ServerConfig           `yaml:"server"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port" default:"7194" validate:"gte=0"`
}

type ClusterConfig struct {
	Metadata        map[string]string   `yaml:"metadata" validate:"dive,keys,required,prometheus_label,endkeys,required"`
	Nodes           NodesProviderConfig `yaml:"nodes" validate:"dive,required"`
	CouchbaseConfig CouchbaseConfig     `yaml:"couchbase_config"`
	MetricsConfig   MetricsConfig       `yaml:"metrics_config"`
}

type ClusterMonitorConfig struct {
	Host     string `yaml:"host" validate:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type CouchbaseConfig struct {
	ManagementPort          int    `default:"18091" yaml:"management_port"`
	Username                string `yaml:"username" validate:"required"`
	Password                string `yaml:"password" validate:"required"` // TODO env var / secret
	IgnoreCertificateErrors bool   `yaml:"ignore_certificate_errors_this_is_insecure"`
}

type MetricsConfig struct {
	ExporterPort int    `yaml:"exporter_port" default:"9091"` // TODO CB7
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
}

type NodesProviderConfig struct {
	Static []string `validate:"required,dive,hostname|ip"`
}

func (npc *NodesProviderConfig) GetNodes() []string {
	return npc.Static
}

func FromYAMLValidate(data []byte) (*Config, error) {
	var result Config
	err := yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	err = cfgValidator.ValidateWithDefaults(&result)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &result, nil
}

func NewDefault() *Config {
	var result Config
	defaults.MustSet(&result)
	return &result
}

func (c Config) ToYAML() []byte {
	val, err := yaml.Marshal(&c)
	if err != nil {
		// All our config should have sensible defaults and validation, so a failed marshaling means we've messed up
		return nil
	}
	return val
}
