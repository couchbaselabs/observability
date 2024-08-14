// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/require"
)

func TestGetLogs(t *testing.T) {
	testHandler := &couchbase.TestHandler{
		PoolsDefault: couchbase.TestPoolsDefaultData{},
		ClusterUUID:  "uuid-0",
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
		NodesReturnCode:  http.StatusOK,
		Buckets:          []couchbase.BucketsEndpointData{},
		BucketReturnCode: http.StatusOK,
		SASLLogs:         "some data here",
		LogName:          "error",
	}

	testHandler.Start(t, true, true)
	defer testHandler.Close()

	mgr := createTestManager(t)
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	require.NoError(t, mgr.store.AddCluster(&values.CouchbaseCluster{
		UUID:       "uuid-0",
		Name:       "",
		Enterprise: true,
		User:       "user",
		Password:   "password",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "Node-1",
				Version:           "7.0.0-0000-enterprise",
				Host:              testHandler.URL(),
				ClusterMembership: "active",
				Status:            "status",
				Services:          []string{"kv"},
			},
		},
	}))

	require.NoError(t, mgr.store.AddCluster(&values.CouchbaseCluster{
		UUID:     "uuid-1",
		Name:     "",
		User:     "user",
		Password: "password",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "Node-1",
				Version:           "7.0.0-0000-community",
				Host:              testHandler.URL(),
				ClusterMembership: "active",
				Status:            "status",
				Services:          []string{"kv"},
			},
		},
	}))

	time.Sleep(100 * time.Millisecond)

	type testCase struct {
		name           string
		uuid           string
		logStatusCode  int
		expectedStatus int
	}

	cases := []testCase{
		{
			name:           "error",
			uuid:           "notFound",
			logStatusCode:  http.StatusOK,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "logNotFound",
			uuid:           "uuid-0",
			logStatusCode:  http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "getLogError",
			uuid:           "uuid-0",
			logStatusCode:  http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "OK",
			uuid:           "uuid-0",
			logStatusCode:  http.StatusOK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "CE",
			uuid:           "uuid-1",
			logStatusCode:  http.StatusOK,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testHandler.LogsReturnCode = tc.logStatusCode

			req, err := http.NewRequest(http.MethodGet,
				fmt.Sprintf("http://localhost:%d/api/v1/clusters/%s/nodes/Node-1/logs/error", mgr.config.HTTPPort,
					tc.uuid), nil)
			require.NoError(t, err)

			req.SetBasicAuth("user", "password")

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()
			require.Equal(t, tc.expectedStatus, res.StatusCode)
		})
	}

	t.Run("nodeNotFound", func(t *testing.T) {
		testHandler.LogsReturnCode = http.StatusOK
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("http://localhost:%d/api/v1/clusters/uuid-0/nodes/Node-7/logs/error", mgr.config.HTTPPort), nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusNotFound, res.StatusCode)
	})
}
