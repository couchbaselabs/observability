package prometheus

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

const (
	managedMarkerComment = "CMOS managed"
)

// BasicAuthConfig holds basic auth settings for scraping targets
type BasicAuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// TLSConfig holds TLS settings for scraping targets
type TLSConfig struct {
	InsecureSkipVerify bool `yaml:"insecure_skip_verify"`
}

// HTTPClientConfig holds the client configuration for HTTP scraping
type HTTPClientConfig struct {
	BasicAuth BasicAuthConfig `yaml:"basic_auth"`
	TLSConfig *TLSConfig      `yaml:"tls_config,omitempty"` // TLS Config is optional
	Scheme    string          `yaml:"scheme"`               // Add scheme (http or https)
}

// ScrapeConfig represents the scraping configuration for a target
type ScrapeConfig struct {
	JobName          string           `yaml:"job_name"`
	MetricsPath      string           `yaml:"metrics_path"`
	HTTPClientConfig HTTPClientConfig `yaml:",inline"` // Inline HTTP client config for correct YAML output
	StaticConfigs    []StaticConfig   `yaml:"static_configs,omitempty"`
}

// StaticConfig holds static scraping targets and labels
type StaticConfig struct {
	Targets []string          `yaml:"targets"`
	Labels  map[string]string `yaml:"labels"`
}

// Configuration represents a Prometheus configuration with custom CMOS scrape configs
type Configuration struct {
	base              *yaml.Node
	baseScrapeConfigs []*yaml.Node
	ScrapeConfigs     []*ScrapeConfig `json:"scrape_configs" yaml:"-"`
}

// UnmarshalYAML unpacks the Prometheus configuration, including custom scrape configs
func (c *Configuration) UnmarshalYAML(value *yaml.Node) error {
	c.base = value
	if value.Tag != "!!map" {
		return fmt.Errorf("invalid yaml structure: value is not a !!map")
	}
	var scrapeConfigsSeq *yaml.Node
	for i, child := range value.Content {
		if child.Tag == "!!str" && child.Value == "scrape_configs" {
			if value.Content[i+1].Tag != "!!seq" {
				return fmt.Errorf("invalid yaml structure: scrape_configs is not a !!seq")
			}
			scrapeConfigsSeq = value.Content[i+1]
			break
		}
	}
	if scrapeConfigsSeq == nil {
		return fmt.Errorf("invalid yaml structure: no scrape_configs")
	}
	c.baseScrapeConfigs = make([]*yaml.Node, 0)
	c.ScrapeConfigs = make([]*ScrapeConfig, 0)
	for _, sc := range scrapeConfigsSeq.Content {
		if sc.Tag == "!!map" && (sc.HeadComment == managedMarkerComment || sc.HeadComment == "# "+managedMarkerComment) {
			var val ScrapeConfig
			if err := sc.Decode(&val); err != nil {
				return fmt.Errorf("couldn't unmarshal ScrapeConfig: %w", err)
			}
			c.ScrapeConfigs = append(c.ScrapeConfigs, &val)
		} else {
			c.baseScrapeConfigs = append(c.baseScrapeConfigs, sc)
		}
	}
	return nil
}

// MarshalYAML marshals the Prometheus configuration, including custom CMOS scrape configs
func (c *Configuration) MarshalYAML() (interface{}, error) {
	// Base it off of the existing base, but replace its scrape_configs, or add some if there aren't any
	scrapeConfigs := new(yaml.Node)
	scrapeConfigs.Kind = yaml.SequenceNode
	scrapeConfigs.Tag = "!!seq"
	scrapeConfigs.Content = append(scrapeConfigs.Content, c.baseScrapeConfigs...)
	for _, sc := range c.ScrapeConfigs {
		node := new(yaml.Node)
		if err := node.Encode(sc); err != nil {
			return nil, fmt.Errorf("failed to marshal ScrapeConfig: %w", err)
		}
		node.HeadComment = managedMarkerComment
		scrapeConfigs.Content = append(scrapeConfigs.Content, node)
	}

	output := c.base
	// Find the scrape_configs entry, or add one if none exists
	var added bool
	if output.Tag != "!!map" {
		return nil, fmt.Errorf("failed to marshal scrape configs - output is not a !!map (" +
			"possibly corrupt internals)")
	}
	for i, child := range output.Content {
		if child.Tag == "!!str" && child.Value == "scrape_configs" && len(output.Content) > i+1 {
			output.Content[i+1] = scrapeConfigs
			added = true
			break
		}
	}
	if !added {
		output.Content = append(
			output.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "scrape_configs",
			},
			scrapeConfigs,
		)
	}
	return output, nil
}
