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
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const baseConfig = `global:
    slack_api_url: test_slack_api_url
route:
    foo: test
    bar: test2
`

func TestUnmarshalSmoke(t *testing.T) {
	var val ConfigFile
	err := yaml.Unmarshal([]byte(baseConfig), &val)
	require.NoError(t, err)
	require.Equal(t, val.Global.SlackAPIURL, "test_slack_api_url")
}

func TestMarshalIdempotent(t *testing.T) {
	var val ConfigFile
	err := yaml.Unmarshal([]byte(baseConfig), &val)
	require.NoError(t, err)
	result, err := yaml.Marshal(&val)
	require.NoError(t, err)
	require.Equal(t, baseConfig, string(result))
}

func TestAddFields(t *testing.T) {
	var val ConfigFile
	err := yaml.Unmarshal([]byte(baseConfig), &val)
	require.NoError(t, err)
	val.Global.SlackAPIURL = "test_added_slack_api_url"
	val.Global.SMTPHello = "added_hello"
	result, err := yaml.Marshal(&val)
	require.NoError(t, err)
	require.Equal(t, `global:
    slack_api_url: test_added_slack_api_url
    smtp_hello: added_hello
route:
    foo: test
    bar: test2
`, string(result))
}