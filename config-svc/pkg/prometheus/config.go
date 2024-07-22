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

package prometheus

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const (
	managedMarkerComment = "CMOS managed"
)

type BasicAuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type HTTPClientConfig struct {
	BasicAuth BasicAuthConfig `yaml:"basic_auth"`
}

type ScrapeConfig struct {
	// The job name to which the job label is set by default.
	JobName          string           `yaml:"job_name"`
	MetricsPath      string           `yaml:"metrics_path"`
	HTTPClientConfig HTTPClientConfig `yaml:",inline"`
	StaticConfigs    []StaticConfig   `yaml:"static_configs,omitempty"`
}

type StaticConfig struct {
	Targets []string          `yaml:"targets"`
	Labels  map[string]string `yaml:"labels"`
}

// Configuration is a subset of prometheus' config struct with special YAML marshal/unmarshal handling.
// Adding a ScrapeConfig to the ScrapeConfigs slice will mean it will be marshaled in the resulting YAML with a special
// marker comment. Unmarshalling the YAML will yield the "added" scrape_configs in ScrapeConfigs, but keep the user's
// scrape_configs intact. In other words, ScrapeConfigs written by CMOS can be later edited by CMOS, but ones written by
// the user will be preserved (and un-editable).
type Configuration struct {
	base              *yaml.Node
	baseScrapeConfigs []*yaml.Node
	ScrapeConfigs     []*ScrapeConfig `json:"scrape_configs" yaml:"-"`
}

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
		// If it has the marker head comment, decode it as a ScrapeConfig struct, otherwise save it in baseScrapeConfigs
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
