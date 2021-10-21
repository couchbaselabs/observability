package metacfg

import (
	"fmt"
	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
	"time"
)

type Config struct {
	Clusters              []ClusterConfig        `yaml:"clusters" default:"[]"`
	ClusterUpdateInterval time.Duration          `default:"1m" yaml:"cluster_update_interval"`
	PrometheusHosts       []string               `yaml:"prometheus_hosts" validate:"dive,url" default:"[]"`
	ClusterMonitorHosts   []ClusterMonitorConfig `yaml:"cluster_monitor_hosts" default:"[]"`
	Immutable             bool                   `yaml:"immutable"`
	Server                ServerConfig           `yaml:"server"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port" default:"7194"`
}

type ClusterConfig struct {
	Metadata        map[string]string   `yaml:"metadata"`
	Nodes           NodesProviderConfig `yaml:"nodes"`
	CouchbaseConfig CouchbaseConfig     `yaml:"couchbase_config"`
	MetricsConfig   MetricsConfig       `yaml:"metrics_config"`
}

type ClusterMonitorConfig struct {
	Host     string `yaml:"host" validate:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type CouchbaseConfig struct {
	ManagementPort int    `default:"18091" yaml:"management_port"`
	Username       string `yaml:"username" validate:"required"`
	Password       string `yaml:"password" validate:"required"` // TODO env var / secret
}

type MetricsConfig struct {
	ExporterPort int    `yaml:"exporter_port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
}

type NodesProviderConfig struct {
	Static []string `validate:"required,dive,hostname"`
}

var cfgValidator = new(Validator)

func FromYAML(data []byte) (*Config, error) {
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
