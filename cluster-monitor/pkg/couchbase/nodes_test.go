// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/netutil"
	"github.com/stretchr/testify/require"
)

func TestClientGetNodeStorage(t *testing.T) {
	var statusCode int
	sample := json.RawMessage(`{"availableStorage":{"hdd":[{"path":"/","sizeKBytes":488347692,"usagePercent":8},{"path":
	"/System/Volumes/Data","sizeKBytes":488347692,"usagePercent": 70}]}}`)

	handlers := make(cbrest.TestHandlers)
	handlers.Add(http.MethodGet, string(NodesSelfEndpoint), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(statusCode, sample, []byte{}, w)
	})

	cluster := cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: true,
		UUID:       "cluster_0",
		Nodes:      cbrest.TestNodes{&cbrest.TestNode{}},
		Handlers:   handlers,
	})
	defer cluster.Close()

	client := getTestClient(t, cluster.URL())

	t.Run("404", func(t *testing.T) {
		statusCode = 404
		_, err := client.GetNodeStorage()
		require.ErrorIs(t, err, values.ErrNotFound)
	})

	t.Run("200", func(t *testing.T) {
		statusCode = 200
		res, _ := client.GetNodeStorage()

		expected := &values.Storage{
			Available: values.AvailableStorage{
				DiskStorage: []values.DiskStorage{
					{
						Path:       "/",
						SizeKBytes: 488347692,
						Usage:      8,
					},
					{
						Path:       "/System/Volumes/Data",
						SizeKBytes: 488347692,
						Usage:      70,
					},
				},
			},
		}

		require.Equal(t, expected, res)
	})
}

func TestClientGetNodesSummary(t *testing.T) {
	handler := &TestHandler{
		ClusterUUID: "cluster_x",
		PoolsDefault: TestPoolsDefaultData{
			ClusterName: "grumpy",
			Nodes: []struct {
				Version string `json:"version"`
			}{
				{
					Version: "7.0.0-0000-enterprise",
				},
			},
			StorageTotals: TestStorageTotals{},
		},
		NodesBytes:       []byte(`"Some error"`),
		BucketReturnCode: http.StatusOK,
	}
	handler.Start(t, false, true)
	defer handler.Close()

	// remove the http:// schema
	noSchemaHost := netutil.TrimSchema(handler.URL())
	host, _, _ := net.SplitHostPort(noSchemaHost)

	type testCase struct {
		name            string
		returnCode      int
		nodes           []TestNode
		expectedSummary values.NodesSummary
	}

	cases := []testCase{
		{
			name:       "403",
			returnCode: http.StatusForbidden,
		},
		{
			name:       "ipv4-hosts",
			returnCode: http.StatusOK,
			expectedSummary: values.NodesSummary{
				{
					NodeUUID:          "a",
					Status:            "healthy",
					Version:           "7.0.0-0000-enterprise",
					ClusterMembership: "active",
					Services:          []string{"n1ql"},
					Host:              fmt.Sprintf("https://%s:19000", host),
					CPUUtil:           1.0,
					SwapTotal:         1000,
					SwapUsed:          0,
					CPUCount:          1,
				},
				{
					NodeUUID:          "b",
					Status:            "warmup",
					Version:           "7.0.0-0000-enterprise",
					ClusterMembership: "active",
					Services:          []string{"kv"},
					Host:              "https://10.10.10.10:19000",
					CPUUtil:           1.0,
					SwapTotal:         1000,
					SwapUsed:          0,
				},
			},
			nodes: []TestNode{
				{
					NodeUUID:          "a",
					Hostname:          noSchemaHost,
					Services:          []string{"n1ql"},
					Version:           "7.0.0-0000-enterprise",
					Status:            "healthy",
					ClusterMembership: "active",
					Ports:             map[string]uint16{"httpsMgmt": 19000},
					SystemStats: SysStats{
						CPU:       1.0,
						SwapTotal: 1000,
						SwapUsed:  0,
					},
					CPUCount: json.RawMessage("1"),
				},
				{
					NodeUUID:          "b",
					Hostname:          "10.10.10.10:9000",
					Services:          []string{"kv"},
					Version:           "7.0.0-0000-enterprise",
					Status:            "warmup",
					ClusterMembership: "active",
					Ports:             map[string]uint16{"httpsMgmt": 19000},
					SystemStats: SysStats{
						CPU:       1.0,
						SwapTotal: 1000,
						SwapUsed:  0,
					},
					CPUCount: json.RawMessage(`"unknown"`),
				},
			},
		},
		{
			name:       "ipv6-hosts",
			returnCode: http.StatusOK,
			expectedSummary: values.NodesSummary{
				{
					NodeUUID:          "a",
					Status:            "healthy",
					Version:           "7.0.0-0000-enterprise",
					ClusterMembership: "active",
					Services:          []string{"n1ql"},
					Host:              "https://[::1]:19000",
					CPUUtil:           1.0,
					SwapTotal:         1000,
					SwapUsed:          0,
				},
				{
					NodeUUID:          "b",
					Status:            "warmup",
					Version:           "7.0.0-0000-enterprise",
					ClusterMembership: "active",
					Services:          []string{"kv"},
					Host:              "https://[aa:bb:ee:dd]:19000",
					CPUUtil:           1.0,
					SwapTotal:         1000,
					SwapUsed:          0,
				},
			},
			nodes: []TestNode{
				{
					NodeUUID:          "a",
					Hostname:          "[::1]:8091",
					Services:          []string{"n1ql"},
					Version:           "7.0.0-0000-enterprise",
					Status:            "healthy",
					ClusterMembership: "active",
					Ports:             map[string]uint16{"httpsMgmt": 19000},
					SystemStats: SysStats{
						CPU:       1.0,
						SwapTotal: 1000,
						SwapUsed:  0,
					},
				},
				{
					NodeUUID:          "b",
					Hostname:          "[aa:bb:ee:dd]:8091",
					Services:          []string{"kv"},
					Version:           "7.0.0-0000-enterprise",
					Status:            "warmup",
					ClusterMembership: "active",
					Ports:             map[string]uint16{"httpsMgmt": 19000},
					SystemStats: SysStats{
						CPU:       1.0,
						SwapTotal: 1000,
						SwapUsed:  0,
					},
				},
			},
		},
		{
			name:       "no-node-uuid",
			returnCode: http.StatusOK,
			expectedSummary: values.NodesSummary{
				{
					NodeUUID:          "https://[::1]:19000",
					Status:            "healthy",
					Version:           "7.0.0-0000-enterprise",
					ClusterMembership: "active",
					Services:          []string{"n1ql"},
					Host:              "https://[::1]:19000",
					CPUUtil:           1.0,
					SwapTotal:         1000,
					SwapUsed:          0,
				},
				{
					NodeUUID:          "https://[aa:bb:ee:dd]:19000",
					Status:            "warmup",
					Version:           "7.0.0-0000-enterprise",
					ClusterMembership: "active",
					Services:          []string{"kv"},
					Host:              "https://[aa:bb:ee:dd]:19000",
					CPUUtil:           1.0,
					SwapTotal:         1000,
					SwapUsed:          0,
				},
			},
			nodes: []TestNode{
				{
					Hostname:          "[::1]:8091",
					Services:          []string{"n1ql"},
					Version:           "7.0.0-0000-enterprise",
					Status:            "healthy",
					ClusterMembership: "active",
					Ports:             map[string]uint16{"httpsMgmt": 19000},
					SystemStats: SysStats{
						CPU:       1.0,
						SwapTotal: 1000,
						SwapUsed:  0,
					},
				},
				{
					Hostname:          "[aa:bb:ee:dd]:8091",
					Services:          []string{"kv"},
					Version:           "7.0.0-0000-enterprise",
					Status:            "warmup",
					ClusterMembership: "active",
					Ports:             map[string]uint16{"httpsMgmt": 19000},
					SystemStats: SysStats{
						CPU:       1.0,
						SwapTotal: 1000,
						SwapUsed:  0,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler.NodesReturnCode = tc.returnCode
			handler.Nodes = tc.nodes

			client := getTestClient(t, handler.URL())

			nodesOut, err := client.GetNodesSummary()

			if tc.returnCode == http.StatusOK && err != nil {
				require.NoError(t, err)
			}

			if tc.returnCode != http.StatusOK {
				return
			}

			require.Equal(t, tc.expectedSummary, nodesOut)
		})
	}
}
