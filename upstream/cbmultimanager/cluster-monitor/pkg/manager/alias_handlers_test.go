// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/require"
)

func TestAddAlias(t *testing.T) {
	type testCase struct {
		name           string
		alias          string
		clusterUUID    string
		expectedStatus int
	}

	cases := []testCase{
		{
			name:           "not-correct-prefix",
			alias:          "aalias",
			clusterUUID:    "uuid-0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "alias-to-long",
			alias:          "a-" + strings.Repeat("1", 100),
			clusterUUID:    "uuid-0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no-cluster-with-that-uuid",
			alias:          "a-1",
			clusterUUID:    "fake",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "no-cluster-uuid",
			alias:          "a-1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid",
			alias:          "a-3",
			clusterUUID:    "uuid-0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "cluster-already-has-alias",
			alias:          "a-7",
			clusterUUID:    "uuid-1",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	mgr := createTestManager(t)
	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	require.NoError(t, mgr.store.AddCluster(&values.CouchbaseCluster{
		UUID:       "uuid-1",
		Alias:      "a-2",
		Enterprise: true,
		Name:       "c0",
		User:       "user",
		Password:   "pass",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "node0",
				Version:           "7.0.0-0000-enterprise",
				Host:              "alpha",
				Status:            "healthy",
				ClusterMembership: "active",
				Services:          []string{"kv"},
			},
		},
		ClusterInfo: &values.ClusterInfo{
			RAMQuota:       9000,
			RAMUsed:        10,
			DiskTotal:      10000,
			DiskUsed:       10,
			DiskUsedByData: 7,
		},
		HeartBeatIssue: values.NoHeartIssue,
		CaCert:         []byte{},
	}))

	require.NoError(t, mgr.store.AddCluster(&values.CouchbaseCluster{
		UUID:       "uuid-0",
		Enterprise: true,
		Name:       "c0",
		User:       "user",
		Password:   "pass",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "node0",
				Version:           "7.0.0-0000-enterprise",
				Host:              "alpha",
				Status:            "healthy",
				ClusterMembership: "active",
				Services:          []string{"kv"},
			},
		},
		ClusterInfo: &values.ClusterInfo{
			RAMQuota:       9000,
			RAMUsed:        10,
			DiskTotal:      10000,
			DiskUsed:       10,
			DiskUsedByData: 7,
		},
		HeartBeatIssue: values.NoHeartIssue,
		CaCert:         []byte{},
	}))

	baseURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/aliases/", mgr.config.HTTPPort)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, baseURL+tc.alias,
				bytes.NewReader([]byte(fmt.Sprintf(`{"cluster_uuid":"%s"}`, tc.clusterUUID))))
			require.NoError(t, err)

			req.SetBasicAuth("user", "password")

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			_ = res.Body.Close()
			require.Equal(t, tc.expectedStatus, res.StatusCode)

			if tc.expectedStatus != http.StatusOK {
				return
			}

			alias, err := mgr.store.GetAlias(tc.alias)
			require.NoError(t, err)
			require.Equal(t, &values.ClusterAlias{Alias: tc.alias, ClusterUUID: tc.clusterUUID}, alias)
		})
	}
}

func TestDeleteAlias(t *testing.T) {
	mgr := createTestManager(t)
	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	require.NoError(t, mgr.store.AddCluster(&values.CouchbaseCluster{
		UUID:       "uuid-1",
		Alias:      "a-1",
		Enterprise: true,
		Name:       "c0",
		User:       "user",
		Password:   "pass",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "node0",
				Version:           "7.0.0-0000-enterprise",
				Host:              "alpha",
				Status:            "healthy",
				ClusterMembership: "active",
				Services:          []string{"kv"},
			},
		},
		ClusterInfo: &values.ClusterInfo{
			RAMQuota:       9000,
			RAMUsed:        10,
			DiskTotal:      10000,
			DiskUsed:       10,
			DiskUsedByData: 7,
		},
		HeartBeatIssue: values.NoHeartIssue,
		CaCert:         []byte{},
	}))

	require.NoError(t, mgr.store.AddCluster(&values.CouchbaseCluster{
		UUID:       "uuid-0",
		Alias:      "a-0",
		Enterprise: true,
		Name:       "c0",
		User:       "user",
		Password:   "pass",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "node0",
				Version:           "7.0.0-0000-enterprise",
				Host:              "alpha",
				Status:            "healthy",
				ClusterMembership: "active",
				Services:          []string{"kv"},
			},
		},
		ClusterInfo: &values.ClusterInfo{
			RAMQuota:       9000,
			RAMUsed:        10,
			DiskTotal:      10000,
			DiskUsed:       10,
			DiskUsedByData: 7,
		},
		HeartBeatIssue: values.NoHeartIssue,
		CaCert:         []byte{},
	}))

	req, err := http.NewRequest(http.MethodDelete,
		fmt.Sprintf("http://127.0.0.1:%d/api/v1/aliases/a-0", mgr.config.HTTPPort), nil)
	require.NoError(t, err)

	req.SetBasicAuth("user", "password")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	_, err = mgr.store.GetAlias("a-0")
	require.ErrorIs(t, err, values.ErrNotFound)

	_, err = mgr.store.GetAlias("a-1")
	require.NoError(t, err)
}
