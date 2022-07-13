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
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const testYaml = `global:
    foo: bar
    qux: quux
scrape_configs:
    - job_name: test
      static_configs:
        - targets:
            - test1
            - test2
          labels:
            foo: bar
            test: label
`

func TestConfigMarshalIdempotent(t *testing.T) {
	var value Configuration
	err := yaml.Unmarshal([]byte(testYaml), &value)
	require.NoError(t, err)

	marshaled, err := yaml.Marshal(&value)
	require.NoError(t, err)
	require.Equal(t, testYaml, string(marshaled))
}

func TestConfigAddNewScrape(t *testing.T) {
	var value Configuration
	err := yaml.Unmarshal([]byte(testYaml), &value)
	require.NoError(t, err)

	value.ScrapeConfigs = append(value.ScrapeConfigs, &ScrapeConfig{
		JobName:     "added",
		MetricsPath: "/metrics",
		StaticConfigs: []StaticConfig{
			{
				Targets: []string{"test"},
			},
		},
	})

	marshaled, err := yaml.Marshal(&value)
	require.NoError(t, err)
	require.Equal(t, testYaml+`    # CMOS managed
    - job_name: added
      metrics_path: /metrics
      basic_auth:
        username: ""
        password: ""
      static_configs:
        - targets:
            - test
          labels: {}
`, string(marshaled))
}

func TestInvalidYAML(t *testing.T) {
	var value Configuration
	err := yaml.Unmarshal([]byte(`foo: bar; invalid`), &value)
	require.Error(t, err)
}

func TestInvalidScrapeConfigs(t *testing.T) {
	var value Configuration
	err := yaml.Unmarshal([]byte(`global:
    foo: bar
    qux: quux
scrape_configs:
    # CMOS managed
    - job_name: 1234
      nonsense: field
      static_configs: {}
`), &value)
	require.Error(t, err, "expected failure to parse %+v", value)
}
