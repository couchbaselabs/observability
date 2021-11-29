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

package alertmanager

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type GlobalContents struct {
	SlackAPIURL     string `yaml:"slack_api_url,omitempty" mapstructure:"slack_api_url"`
	SlackAPIURLFile string `yaml:"slack_api_url_file,omitempty" mapstructure:"slack_api_url_file"`

	SMTPFrom         string `yaml:"smtp_from,omitempty" mapstructure:"smtp_from"`
	SMTPSmarthost    string `yaml:"smtp_smarthost,omitempty" mapstructure:"smtp_smarthost"`
	SMTPHello        string `yaml:"smtp_hello,omitempty" mapstructure:"smtp_hello"`
	SMTPAuthUsername string `yaml:"smtp_auth_username,omitempty" mapstructure:"smtp_auth_username"`
	SMTPAuthPassword string `yaml:"smtp_auth_password,omitempty" mapstructure:"smtp_auth_password"`
	SMTPAuthIdentity string `yaml:"smtp_auth_identity,omitempty" mapstructure:"smtp_auth_identity"`
	SMTPAuthSecret   string `yaml:"smtp_auth_secret,omitempty" mapstructure:"smtp_auth_secret"`
	SMTPRequireTLS   bool   `yaml:"smtp_require_tls,omitempty" mapstructure:"smtp_require_tls"`
}

type Global struct {
	GlobalContents
	base *yaml.Node
}

func mergeYamlMaps(result *yaml.Node, sources ...*yaml.Node) error {
	if result.Kind != yaml.MappingNode {
		return fmt.Errorf("result not a mapping (kind, tag %v %v)", result.Kind, result.Tag)
	}
	for srcIdx, source := range sources {
		if source.Kind != yaml.MappingNode {
			return fmt.Errorf("source %d not a mapping (kind, tag %v %v)", srcIdx, source.Kind, source.Tag)
		}
		// Elements are (!!str, value) pairs, so we go two at a time to catch just the keys
		for i := 0; i < len(source.Content); i = i + 2 {
			// Check if it's already there in the result, if not add it
			srcKey := source.Content[i]
			found := false
			for j := 0; j < len(result.Content); j = j + 2 {
				resKey := result.Content[j]
				if srcKey.Value == resKey.Value {
					result.Content[j+1] = source.Content[i+1]
					found = true
				}
			}
			if !found {
				result.Content = append(
					result.Content,
					srcKey,
					source.Content[i+1],
				)
			}
		}
	}
	return nil
}

func (g *Global) MarshalYAML() (interface{}, error) {
	result := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}
	contents := new(yaml.Node)
	if err := contents.Encode(g.GlobalContents); err != nil {
		return nil, err
	}
	if err := mergeYamlMaps(result, g.base, contents); err != nil {
		return nil, err
	}
	return result, nil
}

func (g *Global) UnmarshalYAML(value *yaml.Node) error {
	if value.Tag != "!!map" {
		return fmt.Errorf("invalid Global structure: value is not a !!map")
	}
	g.base = value
	return value.Decode(&g.GlobalContents)
}

type ConfigFile struct {
	base   *yaml.Node
	Global *Global `yaml:"global"`
}

func (c *ConfigFile) MarshalYAML() (interface{}, error) {
	result := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}
	global := new(yaml.Node)
	if err := global.Encode(c.Global); err != nil {
		return nil, err
	}
	globalWrapper := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "global"},
			global,
		},
	}
	if err := mergeYamlMaps(result, c.base, globalWrapper); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ConfigFile) UnmarshalYAML(value *yaml.Node) error {
	c.base = value
	if value.Tag != "!!map" {
		return fmt.Errorf("invalid ConfigFile structure: value is not a !!map")
	}
	for i, elem := range value.Content {
		if elem.Tag == "!!str" && elem.Value == "global" &&
			len(value.Content) > i+1 && value.Content[i+1].Tag == "!!map" {
			var globalVal Global
			if err := value.Content[i+1].Decode(&globalVal); err != nil {
				return err
			}
			c.Global = &globalVal
		}
	}
	return nil
}
