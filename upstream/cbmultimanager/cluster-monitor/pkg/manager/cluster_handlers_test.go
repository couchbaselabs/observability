// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/manager/mocks"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/auth"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/configuration"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.ConsoleSeparator = " "

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, os.Stdout, zapcore.WarnLevel)

	zap.ReplaceGlobals(zap.New(core))
}

func createTestManager(t *testing.T) *Manager {
	testDir := t.TempDir()
	mgr, err := NewManager(&configuration.Config{
		SQLiteKey:         "password",
		SQLiteDB:          filepath.Join(testDir, "database.sqlite"),
		HTTPPort:          testHTTPPort,
		HTTPSPort:         testHTTPPort + 1,
		MaxWorkers:        1,
		DisableHTTPS:      true,
		EnableAdminAPI:    true,
		EnableClusterAPI:  true,
		EnableExtendedAPI: true,
	})
	require.NoError(t, err)

	password, err := auth.HashPassword("password")
	require.NoError(t, err)

	require.NoError(t, mgr.store.AddUser(&values.User{User: "user", Password: password, Admin: true}))
	mgr.initialized = true

	return mgr
}

func loadTestData(t *testing.T, store storage.Store) {
	// load clusters
	clusters := []values.CouchbaseCluster{
		{
			UUID:       "uuid-0",
			Enterprise: true,
			Alias:      "a-0",
			Name:       "Cluster-0",
			User:       "user",
			Password:   "password",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID:          "Node-0",
					Version:           "7.0.0-0000-enterprise",
					Host:              "http://localhost:9000",
					ClusterMembership: "active",
					Status:            "status",
					Services:          []string{"kv"},
				},
			},
		},
		{
			UUID:       "uuid-1",
			Enterprise: true,
			Name:       "Cluster-1",
			User:       "user",
			Password:   "password",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID:          "Node-1",
					Version:           "7.0.0-0000-enterprise",
					Host:              "http://localhost:8091",
					ClusterMembership: "active",
					Status:            "status",
					Services:          []string{"kv"},
				},
			},
		},
		{
			UUID:     "uuid-2",
			Name:     "CE-cluster",
			User:     "user",
			Password: "password",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID:          "Node-1",
					Version:           "7.0.0-0000-community",
					Host:              "http://localhost:8091",
					ClusterMembership: "active",
					Status:            "status",
					Services:          []string{"kv"},
				},
			},
		},
	}

	for _, cluster := range clusters {
		require.NoError(t, store.AddCluster(&cluster))
	}

	// load checker results
	results := []values.WrappedCheckerResult{
		{
			Cluster: "uuid-0",
			Result: &values.CheckerResult{
				Name:   "checker-0",
				Status: values.GoodCheckerStatus,
				Time:   time.Time{}.UTC(),
			},
		},
		{
			Cluster: "uuid-0",
			Node:    "Node-1",
			Result: &values.CheckerResult{
				Name:   "checker-1",
				Status: values.WarnCheckerStatus,
				Time:   time.Time{}.UTC(),
			},
		},
		{
			Cluster: "uuid-0",
			Result: &values.CheckerResult{
				Name:   "checker-2",
				Status: values.AlertCheckerStatus,
				Time:   time.Time{}.UTC(),
			},
		},
		{
			Cluster: "uuid-0",
			Result: &values.CheckerResult{
				Name:   "checker-3",
				Status: values.InfoCheckerStatus,
				Time:   time.Time{}.UTC(),
			},
		},
		{
			Cluster: "uuid-1",
			Result: &values.CheckerResult{
				Name:   "checker-0",
				Status: values.AlertCheckerStatus,
				Time:   time.Time{}.UTC(),
			},
		},
	}

	for _, result := range results {
		require.NoError(t, store.SetCheckerResult(&result))
	}

	// load dismissal
	require.NoError(t, store.AddDismissal(values.Dismissal{
		Forever:     true,
		Level:       values.ClusterDismissLevel,
		ClusterUUID: "uuid-1",
		ID:          "dismissal-1",
		CheckerName: "checker-0",
	}))
}

func TestGetClusters(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/clusters", mgr.config.HTTPPort)
	t.Run("noClusters", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var clusters []*values.CouchbaseCluster
		require.NoError(t, json.NewDecoder(res.Body).Decode(&clusters))
		require.Len(t, clusters, 0)
	})

	loadTestData(t, mgr.store)

	t.Run("clusters", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)

		var clusters []*values.CouchbaseCluster
		require.NoError(t, json.NewDecoder(res.Body).Decode(&clusters))
		require.Len(t, clusters, 3)

		expectedClusters := []*values.CouchbaseCluster{
			{
				UUID:       "uuid-0",
				Enterprise: true,
				Alias:      "a-0",
				Name:       "Cluster-0",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID:          "Node-0",
						Version:           "7.0.0-0000-enterprise",
						Host:              "http://localhost:9000",
						ClusterMembership: "active",
						Status:            "status",
						Services:          []string{"kv"},
					},
				},
				StatusSummary: &values.ClusterStatusSummary{
					Good:     1,
					Warnings: 1,
					Alerts:   1,
					Info:     1,
				},
			},
			{
				UUID:       "uuid-1",
				Enterprise: true,
				Name:       "Cluster-1",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID:          "Node-1",
						Version:           "7.0.0-0000-enterprise",
						Host:              "http://localhost:8091",
						ClusterMembership: "active",
						Status:            "status",
						Services:          []string{"kv"},
					},
				},
				StatusSummary: &values.ClusterStatusSummary{
					Dismissed: 1,
				},
			},
			{
				UUID: "uuid-2",
				Name: "CE-cluster",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID:          "Node-1",
						Version:           "7.0.0-0000-community",
						Host:              "http://localhost:8091",
						ClusterMembership: "active",
						Status:            "status",
						Services:          []string{"kv"},
					},
				},
			},
		}

		// time gets updated by the store so equate them so they can be compared
		for i, c := range clusters {
			expectedClusters[i].LastUpdate = c.LastUpdate
		}

		require.Equal(t, expectedClusters, clusters)
	})
}

func TestGetCluster(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	loadTestData(t, mgr.store)

	testCm := new(mocks.ClusterManager)
	testCm.On("GetProgress").Return(nil, nil)
	mgr.clusterManagers = NewClusterManagers(map[string]ClusterManager{
		"uuid-0": testCm,
	})

	time.Sleep(100 * time.Millisecond)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/clusters/", mgr.config.HTTPPort)
	t.Run("notFound", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, baseURL+"notFound", nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("found", func(t *testing.T) {
		expected := values.CouchbaseCluster{
			UUID:       "uuid-0",
			Alias:      "a-0",
			Enterprise: true,
			Name:       "Cluster-0",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID:          "Node-0",
					Version:           "7.0.0-0000-enterprise",
					Host:              "http://localhost:9000",
					ClusterMembership: "active",
					Status:            "status",
					Services:          []string{"kv"},
				},
			},
			StatusSummary: &values.ClusterStatusSummary{
				Good:     1,
				Warnings: 1,
				Alerts:   1,
				Info:     1,
			},
		}

		for _, id := range []string{"uuid-0", "a-0"} {
			req, err := http.NewRequest(http.MethodGet, baseURL+id, nil)
			require.NoError(t, err)

			req.SetBasicAuth("user", "password")

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			var cluster values.CouchbaseCluster
			require.NoError(t, json.NewDecoder(res.Body).Decode(&cluster))

			expected.LastUpdate = cluster.LastUpdate
			require.Equal(t, expected, cluster)
		}
	})

	t.Run("found-ce", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, baseURL+"uuid-2", nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		var cluster values.CouchbaseCluster
		require.NoError(t, json.NewDecoder(res.Body).Decode(&cluster))

		expected := values.CouchbaseCluster{
			UUID: "uuid-2",
			Name: "CE-cluster",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID:          "Node-1",
					Version:           "7.0.0-0000-community",
					Host:              "http://localhost:8091",
					ClusterMembership: "active",
					Status:            "status",
					Services:          []string{"kv"},
				},
			},
		}

		expected.LastUpdate = cluster.LastUpdate
		require.Equal(t, expected, cluster)
	})
}

func TestGetClusterNode(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	loadTestData(t, mgr.store)

	time.Sleep(100 * time.Millisecond)

	t.Run("found", func(t *testing.T) {
		url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/clusters/uuid-0/node/Node-0", mgr.config.HTTPPort)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		var node values.NodeSummary
		require.NoError(t, json.NewDecoder(res.Body).Decode(&node))

		require.Equal(t, values.NodeSummary{
			NodeUUID:          "Node-0",
			Version:           "7.0.0-0000-enterprise",
			Host:              "http://localhost:9000",
			ClusterMembership: "active",
			Status:            "status",
			Services:          []string{"kv"},
		}, node)
	})

	t.Run("notFound", func(t *testing.T) {
		url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/clusters/uuid-0/node/Not-Found", mgr.config.HTTPPort)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusNotFound, res.StatusCode)
	})
}

func TestDeleteCluster(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	loadTestData(t, mgr.store)

	time.Sleep(100 * time.Millisecond)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/clusters/", mgr.config.HTTPPort)
	t.Run("notFound", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, baseURL+"notFound", nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)

		// confirm nothing got deleted
		clusters, err := mgr.store.GetClusters(false, false)
		require.NoError(t, err)
		require.Len(t, clusters, 3)
	})

	t.Run("delete", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, baseURL+"uuid-1", nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)

		// confirm deletion
		clusters, err := mgr.store.GetClusters(false, false)
		require.NoError(t, err)
		require.Len(t, clusters, 2)
		require.Equal(t, "uuid-0", clusters[0].UUID)

		// confirm no results for deleted cluster
		results, err := mgr.store.GetCheckerResult(values.CheckerSearch{})
		require.NoError(t, err)

		require.Len(t, results, 4, "expected only 4 results after deleting one of the clusters")
		for _, res := range results {
			require.Equal(t, "uuid-0", res.Cluster)
		}
	})

	t.Run("delete-by-alias", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, baseURL+"a-0", nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)

		// confirm deletion
		clusters, err := mgr.store.GetClusters(false, false)
		require.NoError(t, err)
		require.Len(t, clusters, 1)
		require.Equal(t, "uuid-2", clusters[0].UUID)

		// confirm no results for deleted cluster
		results, err := mgr.store.GetCheckerResult(values.CheckerSearch{})
		require.NoError(t, err)
		require.Len(t, results, 0, "expected 0 results after deleting one of the clusters")
	})
}

type clusterAddOrUpdateTest struct {
	name            string
	uuid            string
	requestBody     []byte
	expectedStatus  int
	expectedCluster *values.CouchbaseCluster
	clientFail      bool
}

func TestAddCECluster(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	time.Sleep(100 * time.Millisecond)

	testHandler := &couchbase.TestHandler{
		ClusterUUID:  "uuid-0",
		PoolsDefault: couchbase.TestPoolsDefaultData{},
		Nodes: []couchbase.TestNode{
			{
				NodeUUID:          "node-0",
				Hostname:          "127.0.0.1:9000",
				Services:          []string{"kv", "backup"},
				Version:           "7.0.0-0000-community",
				Status:            "healthy",
				ClusterMembership: "active",
				Ports:             map[string]uint16{"httpsMgmt": 19000},
			},
		},
		Buckets:            []couchbase.BucketsEndpointData{},
		RemoteClusters:     []couchbase.RemoteClustersEndpointData{},
		RemoteClustersCode: http.StatusOK,
		BucketReturnCode:   http.StatusOK,
		NodesReturnCode:    http.StatusOK,
	}

	testHandler.Start(t, true, false)
	defer testHandler.Close()

	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("http://localhost:%d/api/v1/clusters", mgr.config.HTTPPort),
		bytes.NewReader([]byte(fmt.Sprintf(`{"password":"pass","user":"user","host":"%s"}`, testHandler.URL()))))
	require.NoError(t, err)

	req.SetBasicAuth("user", "password")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)

	expectedCluster := &values.CouchbaseCluster{
		UUID:     "uuid-0",
		User:     "user",
		Password: "pass",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "node-0",
				Host:              "http://127.0.0.1:9000",
				Services:          []string{"kv", "backup"},
				Status:            "healthy",
				ClusterMembership: "active",
				Version:           "7.0.0-0000-community",
			},
		},
		BucketsSummary: values.BucketsSummary{},
		RemoteClusters: values.RemoteClusters{},
		ClusterInfo:    &values.ClusterInfo{},
		CaCert:         []byte{},
	}

	cluster, err := mgr.store.GetCluster(expectedCluster.UUID, true)
	require.NoError(t, err)

	expectedCluster.LastUpdate = cluster.LastUpdate
	require.Equal(t, expectedCluster, cluster)
}

func TestAddNewClusterWithAlias(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	time.Sleep(100 * time.Millisecond)

	testHandler := &couchbase.TestHandler{
		ClusterUUID:  "uuid-0",
		PoolsDefault: couchbase.TestPoolsDefaultData{},
		Nodes: []couchbase.TestNode{
			{
				NodeUUID:          "node-0",
				Hostname:          "127.0.0.1:9000",
				Services:          []string{"kv", "backup"},
				Version:           "7.0.0-0000-enterprise",
				Status:            "healthy",
				ClusterMembership: "active",
				Ports:             map[string]uint16{"httpsMgmt": 19000},
			},
		},
		Buckets:            []couchbase.BucketsEndpointData{},
		RemoteClusters:     []couchbase.RemoteClustersEndpointData{},
		BucketReturnCode:   http.StatusOK,
		RemoteClustersCode: http.StatusOK,
		NodesReturnCode:    http.StatusOK,
	}

	testHandler.Start(t, true, true)
	defer testHandler.Close()

	t.Run("invalid-alias", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("http://localhost:%d/api/v1/clusters", mgr.config.HTTPPort),
			bytes.NewReader([]byte(fmt.Sprintf(`{"password":"pass","user":"user","host":"%s","alias":"sarandonga"}`,
				testHandler.URL()))))
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("valid-alias", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("http://localhost:%d/api/v1/clusters", mgr.config.HTTPPort),
			bytes.NewReader([]byte(fmt.Sprintf(`{"password":"pass","user":"user","host":"%s","alias":"a-sarandonga"}`,
				testHandler.URL()))))
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)

		alias, err := mgr.store.GetAlias("a-sarandonga")
		require.NoError(t, err)
		require.Equal(t, &values.ClusterAlias{Alias: "a-sarandonga", ClusterUUID: "uuid-0"}, alias)
	})
}

func TestAddNewCluster(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	time.Sleep(100 * time.Millisecond)

	testHandler := &couchbase.TestHandler{
		ClusterUUID:  "uuid-0",
		PoolsDefault: couchbase.TestPoolsDefaultData{},
		Nodes: []couchbase.TestNode{
			{
				NodeUUID:          "node-0",
				Hostname:          "127.0.0.1:9000",
				Services:          []string{"kv", "backup"},
				Version:           "7.0.0-0000-enterprise",
				Status:            "healthy",
				ClusterMembership: "active",
				Ports:             map[string]uint16{"httpsMgmt": 19000},
			},
		},
		Buckets:            []couchbase.BucketsEndpointData{},
		RemoteClusters:     []couchbase.RemoteClustersEndpointData{},
		BucketReturnCode:   http.StatusOK,
		RemoteClustersCode: http.StatusOK,
	}

	testHandler.Start(t, true, true)
	defer testHandler.Close()

	cases := []clusterAddOrUpdateTest{
		{
			name:           "invalidJSON",
			requestBody:    []byte(`{"a":b,}`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "noHost",
			requestBody:    []byte(`{"user":"u","password":"pass"}`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "noPassword",
			requestBody:    []byte(fmt.Sprintf(`{"user":"u","host":"%s"}`, testHandler.URL())),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "noUser",
			requestBody:    []byte(fmt.Sprintf(`{"password":"pass","host":"%s"}`, testHandler.URL())),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalidHost",
			requestBody:    []byte(`{"password":"pass","host":"file:///default.com:8091:40"}`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "client401",
			requestBody:    []byte(fmt.Sprintf(`{"password":"pass","user":"user","host":"%s"}`, testHandler.URL())),
			expectedStatus: http.StatusInternalServerError,
			clientFail:     true,
		},
		{
			name:           "OK",
			requestBody:    []byte(fmt.Sprintf(`{"password":"pass","user":"user","host":"%s"}`, testHandler.URL())),
			expectedStatus: http.StatusOK,
			expectedCluster: &values.CouchbaseCluster{
				UUID:       "uuid-0",
				Enterprise: true,
				User:       "user",
				Password:   "pass",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID:          "node-0",
						Host:              "https://127.0.0.1:19000",
						Services:          []string{"kv", "backup"},
						Status:            "healthy",
						ClusterMembership: "active",
						Version:           "7.0.0-0000-enterprise",
					},
				},
				BucketsSummary: values.BucketsSummary{},
				RemoteClusters: values.RemoteClusters{},
				ClusterInfo:    &values.ClusterInfo{},
				CaCert:         []byte{},
			},
		},
		{
			name:           "duplicateCluster",
			requestBody:    []byte(fmt.Sprintf(`{"password":"pass","user":"user","host":"%s"}`, testHandler.URL())),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testHandler.NodesReturnCode = http.StatusOK
			if tc.clientFail {
				testHandler.NodesReturnCode = http.StatusUnauthorized
			}

			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d/api/v1/clusters",
				mgr.config.HTTPPort), bytes.NewReader(tc.requestBody))
			require.NoError(t, err)

			req.SetBasicAuth("user", "password")

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			require.Equal(t, tc.expectedStatus, res.StatusCode)

			if tc.expectedCluster == nil {
				return
			}

			cluster, err := mgr.store.GetCluster(tc.expectedCluster.UUID, true)
			require.NoError(t, err)

			tc.expectedCluster.LastUpdate = cluster.LastUpdate
			require.Equal(t, tc.expectedCluster, cluster)
		})
	}
}

func TestUpdateClusterInfo(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	testCm := new(mocks.ClusterManager)
	testCm.On("UpdateClusterInfo", mock.Anything).Return(nil)
	mgr.clusterManagers = NewClusterManagers(map[string]ClusterManager{
		"uuid-0": testCm,
	})

	loadTestData(t, mgr.store)

	time.Sleep(100 * time.Millisecond)

	testHandler := &couchbase.TestHandler{
		ClusterUUID:  "uuid-0",
		PoolsDefault: couchbase.TestPoolsDefaultData{ClusterName: "NewName"},
		Nodes: []couchbase.TestNode{
			{
				NodeUUID:          "node-0",
				Hostname:          "127.0.0.1:9000",
				Services:          []string{"kv", "backup"},
				Version:           "7.0.0-0000-enterprise",
				Status:            "healthy",
				ClusterMembership: "active",
				Ports:             map[string]uint16{"httpsMgmt": 19000},
			},
		},
		Buckets:          []couchbase.BucketsEndpointData{},
		BucketReturnCode: http.StatusOK,
	}

	testHandler.Start(t, true, true)
	defer testHandler.Close()

	cases := []clusterAddOrUpdateTest{
		{
			name:           "invalidJSON",
			requestBody:    []byte(`{"a":b,}`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalidHost",
			requestBody:    []byte(`{"password":"pass","host":"file:///default.com:8091:40"}`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "client401",
			requestBody:    []byte(fmt.Sprintf(`{"password":"pass","user":"user","host":"%s"}`, testHandler.URL())),
			expectedStatus: http.StatusInternalServerError,
			clientFail:     true,
		},
		{
			name:           "clusterNotFound",
			uuid:           "notFoundCluster",
			requestBody:    []byte(fmt.Sprintf(`{"password":"pass","user":"user","host":"%s"}`, testHandler.URL())),
			expectedStatus: http.StatusNotFound,
			clientFail:     true,
		},
		{
			name:           "uuidsDontMatch",
			uuid:           "uuid-1",
			requestBody:    []byte(fmt.Sprintf(`{"password":"pass","user":"user","host":"%s"}`, testHandler.URL())),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "OK",
			requestBody:    []byte(fmt.Sprintf(`{"password":"pass1","user":"user1","host":"%s"}`, testHandler.URL())),
			expectedStatus: http.StatusOK,
			expectedCluster: &values.CouchbaseCluster{
				UUID:       "uuid-0",
				Alias:      "a-0",
				Enterprise: true,
				Name:       "NewName",
				User:       "user1",
				Password:   "pass1",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID:          "node-0",
						Host:              "https://127.0.0.1:19000",
						Services:          []string{"kv", "backup"},
						Status:            "healthy",
						ClusterMembership: "active",
						Version:           "7.0.0-0000-enterprise",
					},
				},
				CaCert: []byte{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testHandler.NodesReturnCode = http.StatusOK
			if tc.clientFail {
				testHandler.NodesReturnCode = http.StatusUnauthorized
			}

			uuid := "uuid-0"
			if tc.uuid != "" {
				uuid = tc.uuid
			}

			req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("http://localhost:%d/api/v1/clusters/%s",
				mgr.config.HTTPPort, uuid), bytes.NewReader(tc.requestBody))
			require.NoError(t, err)

			req.SetBasicAuth("user", "password")

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			require.Equal(t, tc.expectedStatus, res.StatusCode)

			if tc.expectedCluster == nil {
				return
			}

			cluster, err := mgr.store.GetCluster(tc.expectedCluster.UUID, true)
			require.NoError(t, err)

			tc.expectedCluster.LastUpdate = cluster.LastUpdate
			require.Equal(t, tc.expectedCluster, cluster)
		})
	}
}
