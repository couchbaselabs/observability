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
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const baseConfig = `global:
    slack_api_url: test_slack_api_url
route:
    foo: test
    bar: test2
`

func TestGetAlertsConfiguration(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	server, err := NewServer(logger, "/", true)
	require.NoError(t, err)

	tmpdir := t.TempDir()
	configPath := path.Join(tmpdir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(baseConfig), 0o644))
	require.NoError(t, os.Setenv("ALERTMANAGER_CONFIG_FILE", configPath))

	r := httptest.NewRequest(http.MethodGet, "/api/v1/alertsConfiguration", nil)
	w := httptest.NewRecorder()
	ctx := server.echo.NewContext(r, w)

	require.NoError(t, server.GetAlertsConfiguration(ctx))

	require.JSONEq(t, `{"slack": {"webhookURL": "test_slack_api_url"}, "email": {"from": "", "hello": "", "host": "",
"identity": "", "password": "", "requireTLS": false, "username": "", "secret": ""}}`,
		w.Body.String())
}

func TestPutAlertsConfiguration(t *testing.T) {
	server, configPath := setupTestServer(t)

	r := httptest.NewRequest(http.MethodPut, "/api/v1/alertsConfiguration", bytes.NewBufferString(`{
		"slack": {
			"webhookURL": "written"
		}
	}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx := server.echo.NewContext(r, w)

	require.NoError(t, server.PutAlertsConfiguration(ctx))

	contents, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.Equal(t, `global:
    slack_api_url: written
route:
    foo: test
    bar: test2
`, string(contents))
}

func setupTestServer(t *testing.T) (*Server, string) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	server, err := NewServer(logger, "/", true)
	require.NoError(t, err)

	tmpdir := t.TempDir()
	configPath := path.Join(tmpdir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(baseConfig), 0o644))
	require.NoError(t, os.Setenv("ALERTMANAGER_CONFIG_FILE", configPath))
	return server, configPath
}
