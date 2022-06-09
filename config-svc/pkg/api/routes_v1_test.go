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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/cbvalue"
	"github.com/couchbaselabs/observability/config-svc/pkg/couchbase"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const basePromConfig = `global:
    scrapeInterval: 30s
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

func TestPostClustersAdd(t *testing.T) {
	t.Run("CreateConfig", func(t *testing.T) {
		promCfgPath, testCluster := setupForTest(t, cbrest.TestClusterOptions{
			Handlers: map[string]http.HandlerFunc{
				"GET:/pools/default": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(&couchbase.PoolsDefault{
						ClusterName: "Test Cluster",
						Nodes: []couchbase.Node{
							{
								Hostname: "test",
								Version:  cbvalue.Version7_0_0,
							},
						},
					})
				},
			},
		})
		defer testCluster.Close()

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/add", bytes.NewReader([]byte(fmt.Sprintf(`{
			"hostname": "%s",
			"couchbaseConfig": {
				"username": "Administrator",
				"password": "asdasd",
				"managementPort": %d
			}
		}`, testCluster.Hostname(), testCluster.Port()))))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		ctx := e.NewContext(req, rec)
		h := &Server{
			baseLogger: zap.NewNop(),
			logger:     zap.NewNop(),
			echo:       e,
			production: true,
		}

		err := h.PostClustersAdd(ctx)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, rec.Code)

		result, err := os.ReadFile(promCfgPath)
		require.NoError(t, err)
		require.Equal(t, basePromConfig+`    # CMOS managed
    - job_name: couchbase-server-managed-1
      basic_auth:
        username: Administrator
        password: asdasd
      static_configs:
        - targets:
            - test:8091
          labels:
            cluster_name: Test Cluster
`, string(result))
	})

	t.Run("CreateConfigCustomMetricsPort", func(t *testing.T) {
		promCfgPath, testCluster := setupForTest(t, cbrest.TestClusterOptions{
			Handlers: map[string]http.HandlerFunc{
				"GET:/pools/default": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(&couchbase.PoolsDefault{
						ClusterName: "Test Cluster",
						Nodes: []couchbase.Node{
							{
								Hostname: "test",
								Version:  cbvalue.Version6_6_0,
							},
						},
					})
				},
			},
		})
		defer testCluster.Close()

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/add", bytes.NewReader([]byte(fmt.Sprintf(`{
			"hostname": "%s",
			"couchbaseConfig": {
				"username": "Administrator",
				"password": "asdasd",
				"managementPort": %d
			},
			"metricsConfig": {
				"metricsPort": 9999
			}
		}`, testCluster.Hostname(), testCluster.Port()))))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		ctx := e.NewContext(req, rec)
		h := &Server{
			baseLogger: zap.NewNop(),
			logger:     zap.NewNop(),
			echo:       e,
			production: true,
		}

		err := h.PostClustersAdd(ctx)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, rec.Code)

		result, err := os.ReadFile(promCfgPath)
		require.NoError(t, err)
		require.Equal(t, basePromConfig+`    # CMOS managed
    - job_name: couchbase-server-managed-1
      basic_auth:
        username: ""
        password: ""
      static_configs:
        - targets:
            - test:9999
          labels:
            cluster_name: Test Cluster
`, string(result))
	})
}

func setupForTest(t *testing.T, opts cbrest.TestClusterOptions) (string, *cbrest.TestCluster) {
	testDir := t.TempDir()
	promCfg := filepath.Join(testDir, "prometheus.yml")
	err := os.WriteFile(promCfg, []byte(basePromConfig), 0o666)
	require.NoError(t, err)
	require.NoError(t, os.Setenv("PROMETHEUS_CONFIG_FILE", promCfg))

	testCluster := cbrest.NewTestCluster(t, opts)
	fmt.Println(testCluster.URL())
	return promCfg, testCluster
}
