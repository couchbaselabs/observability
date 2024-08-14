// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package status

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

type nodeTestCase struct {
	name     string
	expected []*values.WrappedCheckerResult
	nodes    values.NodesSummary
}

func TestOneServicePerNodeCheck(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
	}

	cases := []nodeTestCase{
		{
			name: "1-node-1-service",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Services: []string{"kv"},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckOneServicePerNode,
						Status: values.GoodCheckerStatus,
						Time:   cluster.LastUpdate,
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
		},
		{
			name: "1-node-1-service-1-node-2-services",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Services: []string{"kv"},
				},
				{
					NodeUUID: "N1",
					Services: []string{"kv", "backup"},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckOneServicePerNode,
						Status: values.GoodCheckerStatus,
						Time:   cluster.LastUpdate,
					},
					Cluster: "C0",
					Node:    "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckOneServicePerNode,
						Status: values.InfoCheckerStatus,
						Value:  []byte(`{"node_uuid":"N1","services":["kv","backup"]}`),
						Time:   cluster.LastUpdate,
					},
					Cluster: "C0",
					Node:    "N1",
				},
			},
		},
		{
			name: "1-node-3-service-1-node-2-services",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Services: []string{"kv", "backup", "index"},
				},
				{
					NodeUUID: "N1",
					Services: []string{"kv", "backup"},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckOneServicePerNode,
						Status: values.InfoCheckerStatus,
						Time:   cluster.LastUpdate,
						Value:  []byte(`{"node_uuid":"N0","services":["kv","backup","index"]}`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckOneServicePerNode,
						Status: values.InfoCheckerStatus,
						Value:  []byte(`{"node_uuid":"N1","services":["kv","backup"]}`),
						Time:   cluster.LastUpdate,
					},
					Cluster: "C0",
					Node:    "N1",
				},
			},
		},
	}

	runNodeTestCases(t, *cluster, cases, oneServicePerNodeCheck)
}

func TestSupportedVersionCheck(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "uuid-0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
	}

	cases := []nodeTestCase{
		{
			name: "supported",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSupportedVersion,
						Time:   cluster.LastUpdate,
						Value:  []byte(`"6.6.1-9213-enterprise"`),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "uuid-0",
					Node:    "node-0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "node-0",
					Version:  "6.6.1-9213-enterprise",
				},
			},
		},
		{
			name: "unknown",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSupportedVersion,
						Time:   cluster.LastUpdate,
						Status: values.InfoCheckerStatus,
					},
					Cluster: "uuid-0",
					Node:    "node-0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "node-0",
					Version:  "3.0.0-0000-enterprise",
				},
			},
		},
		{
			name: "unsupported-EOM",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSupportedVersion,
						Time:   cluster.LastUpdate,
						Value:  []byte(`"6.0.1-2037-enterprise"`),
						Status: values.InfoCheckerStatus,
					},
					Cluster: "uuid-0",
					Node:    "node-0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "node-0",
					Version:  "6.0.1-2037-enterprise",
				},
			},
		},
		{
			name: "unsupported-EOS",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSupportedVersion,
						Time:   cluster.LastUpdate,
						Value:  []byte(`"5.1.3-6210-enterprise"`),
						Status: values.InfoCheckerStatus,
					},
					Cluster: "uuid-0",
					Node:    "node-0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "node-0",
					Version:  "5.1.3-6210-enterprise",
				},
			},
		},
		{
			name: "mixed-supported",
			nodes: values.NodesSummary{
				{
					NodeUUID: "node-0",
					Version:  "6.0.1-2037-enterprise",
				},
				{
					NodeUUID: "node-1",
					Version:  "6.6.1-9213-enterprise",
				},
				{
					NodeUUID: "node-3",
					Version:  "5.1.3-6210-enterprise",
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSupportedVersion,
						Time:   cluster.LastUpdate,
						Value:  []byte(`"6.0.1-2037-enterprise"`),
						Status: values.InfoCheckerStatus,
					},
					Cluster: "uuid-0",
					Node:    "node-0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSupportedVersion,
						Value:  []byte(`"6.6.1-9213-enterprise"`),
						Time:   cluster.LastUpdate,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "uuid-0",
					Node:    "node-1",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSupportedVersion,
						Time:   cluster.LastUpdate,
						Status: values.InfoCheckerStatus,
						Value:  []byte(`"5.1.3-6210-enterprise"`),
					},
					Cluster: "uuid-0",
					Node:    "node-3",
				},
			},
		},
	}

	runNodeTestCases(t, *cluster, cases, supportedVersionCheck)
}

func TestUnhealthyNodesCheck(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
	}

	cases := []nodeTestCase{
		{
			name: "healthy",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckUnhealthyNode,
						Status: values.GoodCheckerStatus,
						Time:   cluster.LastUpdate,
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID:          "N0",
					Status:            "healthy",
					ClusterMembership: "active",
				},
			},
		},
		{
			name: "healthy-inactive",
			nodes: values.NodesSummary{
				{
					NodeUUID:          "N0",
					Status:            "healthy",
					ClusterMembership: "active",
				},
				{
					NodeUUID:          "N1",
					Status:            "healthy",
					ClusterMembership: "inactive",
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckUnhealthyNode,
						Status: values.GoodCheckerStatus,
						Time:   cluster.LastUpdate,
					},
					Cluster: "C0",
					Node:    "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckUnhealthyNode,
						Status: values.WarnCheckerStatus,
						Time:   cluster.LastUpdate,
						Value:  []byte(`{"node_uuid":"N1","status":"healthy","cluster_membership":"inactive"}`),
					},
					Cluster: "C0",
					Node:    "N1",
				},
			},
		},
		{
			name: "unhealthy-inactive",
			nodes: values.NodesSummary{
				{
					NodeUUID:          "N0",
					Status:            "unhealthy",
					ClusterMembership: "active",
				},
				{
					NodeUUID:          "N1",
					Status:            "healthy",
					ClusterMembership: "inactive",
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckUnhealthyNode,
						Status: values.WarnCheckerStatus,
						Time:   cluster.LastUpdate,
						Value:  []byte(`{"node_uuid":"N0","status":"unhealthy","cluster_membership":"active"}`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckUnhealthyNode,
						Status: values.WarnCheckerStatus,
						Time:   cluster.LastUpdate,
						Value:  []byte(`{"node_uuid":"N1","status":"healthy","cluster_membership":"inactive"}`),
					},
					Cluster: "C0",
					Node:    "N1",
				},
			},
		},
	}

	runNodeTestCases(t, *cluster, cases, unhealthyNodesCheck)
}

func TestNonGABuildTest(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
	}

	cases := []nodeTestCase{
		{
			name: "good",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckNonGABuild,
						Time:   cluster.LastUpdate,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "5.5.4-4338-enterprise",
				},
			},
		},
		{
			name: "warn",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckNonGABuild,
						Time:   cluster.LastUpdate,
						Status: values.InfoCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "6.0.0-0000-enterprise",
				},
			},
		},
		{
			name: "mixed",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckNonGABuild,
						Time:   cluster.LastUpdate,
						Status: values.InfoCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckNonGABuild,
						Time:   cluster.LastUpdate,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N1",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "6.0.0-0000-enterprise",
				},
				{
					NodeUUID: "N1",
					Version:  "5.5.4-4338-enterprise",
				},
			},
		},
	}

	runNodeTestCases(t, *cluster, cases, nonGABuildCheck)
}

func TestNodeSwapUsage(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
	}

	cases := []nodeTestCase{
		{
			name: "good",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckNodeSwapUsage,
						Time:   cluster.LastUpdate,
						Status: values.GoodCheckerStatus,
						Value:  []byte(`0`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID:  "N0",
					SwapTotal: 100,
				},
			},
		},
		{
			name: "warning",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckNodeSwapUsage,
						Time:   cluster.LastUpdate,
						Status: values.WarnCheckerStatus,
						Value:  []byte(`20`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID:  "N0",
					SwapUsed:  20,
					SwapTotal: 100,
				},
			},
		},
		{
			name: "alert",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckNodeSwapUsage,
						Time:   cluster.LastUpdate,
						Status: values.AlertCheckerStatus,
						Value:  []byte(`99`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID:  "N0",
					SwapUsed:  99,
					SwapTotal: 100,
				},
			},
		},
		{
			name: "zeroSwap",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckNodeSwapUsage,
						Time:   cluster.LastUpdate,
						Status: values.GoodCheckerStatus,
						Value:  []byte(`0`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID:  "N0",
					SwapUsed:  0,
					SwapTotal: 0,
				},
			},
		},
	}

	runNodeTestCases(t, *cluster, cases, nodeSwapUsageCheck)
}

func TestBelowMinMemCheck(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
	}
	cases := []nodeTestCase{
		{
			name: "good",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBelowMinMem,
						Time:   cluster.LastUpdate,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					MemTotal: 1000000000000,
				},
			},
		},
		{
			name: "info",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBelowMinMem,
						Time:   cluster.LastUpdate,
						Status: values.InfoCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					MemTotal: 100,
				},
			},
		},
	}
	runNodeTestCases(t, *cluster, cases, belowMinMemCheck)
}

type nodeSelfTestCase struct {
	name     string
	expected []*values.WrappedCheckerResult
	storage  *values.Storage
	cluster  *values.CouchbaseCluster
}

func TestNodeDiskSpace(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
		NodesSummary: values.NodesSummary{
			{
				NodeUUID: "N0",
			},
		},
	}

	cases := []nodeSelfTestCase{
		{
			name: "good",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckNodeDiskSpace,
						Time:   cluster.LastUpdate,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			storage: &values.Storage{
				Available: values.AvailableStorage{
					DiskStorage: []values.DiskStorage{
						{
							Usage: 10,
							Path:  "/default",
						},
					},
				},
			},
		},
		{
			name: "bad",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckNodeDiskSpace,
						Time:   cluster.LastUpdate,
						Status: values.WarnCheckerStatus,
						Value:  []byte(`{"disk":["/default"]}`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			storage: &values.Storage{
				Available: values.AvailableStorage{
					DiskStorage: []values.DiskStorage{
						{
							Usage: 100,
							Path:  "/default",
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for i, node := range cluster.NodesSummary {
				results := nodeDiskSpaceCheck(cluster, tc.storage, node)
				wrappedResultMustMatch(tc.expected[i], results, tc.expected[i].Result.Status != values.GoodCheckerStatus,
					false, t)
			}
		})
	}
}

func TestSharedFileSystemsCheck(t *testing.T) {
	updateTime := time.Now().UTC()
	commonTestCluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: updateTime,
		NodesSummary: values.NodesSummary{
			{
				NodeUUID: "N0",
				Services: []string{"kv", "n1ql", "index", "fts", "cbas", "eventing"},
			},
		},
	}

	cases := []nodeSelfTestCase{
		{
			name: "good",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSharedFilesystems,
						Time:   commonTestCluster.LastUpdate,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			storage: &values.Storage{
				Available: values.AvailableStorage{
					DiskStorage: []values.DiskStorage{
						{
							Path: "/default",
						},
						{
							Path: "/default2",
						},
					},
				},
				NodeStorage: values.NodeStorageSet{
					HDD: []values.StorageConfig{
						{
							Path:      "/default",
							IndexPath: "/default2",
						},
					},
				},
			},
		},
		{
			name: "bad",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSharedFilesystems,
						Time:   commonTestCluster.LastUpdate,
						Status: values.InfoCheckerStatus,
						Value:  []byte(`["Services on /default: Data, Index."]`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			storage: &values.Storage{
				Available: values.AvailableStorage{
					DiskStorage: []values.DiskStorage{
						{
							Usage: 0,
							Path:  "/default",
						},
					},
				},
				NodeStorage: values.NodeStorageSet{
					HDD: []values.StorageConfig{
						{
							Path:      "/default/kv",
							IndexPath: "/default/gsi",
						},
					},
				},
			},
		},
		{
			name: "regression-windows",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSharedFilesystems,
						Time:   updateTime,
						Status: values.InfoCheckerStatus,
						Value:  []byte(`["Services on C:\\: Analytics, Data, Eventing, Index."]`),
					},
					Cluster: "WinC",
					Node:    "WinN",
				},
			},
			cluster: &values.CouchbaseCluster{
				UUID: "WinC",
				Name: "WinC",
				NodesSummary: []values.NodeSummary{
					{
						NodeUUID: "WinN",
						OS:       "win64",
						Services: []string{"kv", "n1ql", "index", "fts", "cbas", "eventing"},
					},
				},
				LastUpdate: updateTime,
			},
			storage: &values.Storage{
				Available: values.AvailableStorage{
					DiskStorage: []values.DiskStorage{
						{
							Path:       "C:\\",
							SizeKBytes: 0,
							Usage:      5,
						},
					},
				},
				NodeStorage: values.NodeStorageSet{
					HDD: []values.StorageConfig{
						{
							Path:         "c:/Program Files/Couchbase/Server/var/lib/couchbase/data",
							IndexPath:    "c:/Program Files/Couchbase/Server/var/lib/couchbase/data",
							CBASDirs:     []string{"c:/Program Files/Couchbase/Server/var/lib/couchbase/data"},
							EventingPath: "c:/Program Files/Couchbase/Server/var/lib/couchbase/data",
						},
					},
				},
			},
		},
		{
			name: "regression-slashRoot",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSharedFilesystems,
						Time:   commonTestCluster.LastUpdate,
						Status: values.InfoCheckerStatus,
						Value:  []byte(`["Services on /: Analytics, Data, Eventing, Index."]`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			storage: &values.Storage{
				Available: values.AvailableStorage{
					DiskStorage: []values.DiskStorage{
						{
							Path:       "/",
							SizeKBytes: 0,
							Usage:      5,
						},
					},
				},
				NodeStorage: values.NodeStorageSet{
					HDD: []values.StorageConfig{
						{
							Path:         "/Users/cbuser/Library/Application Support/Couchbase/var/lib/couchbase/data",
							IndexPath:    "/Users/cbuser/Library/Application Support/Couchbase/var/lib/couchbase/data",
							CBASDirs:     []string{"/Users/cbuser/Library/Application Support/Couchbase/var/lib/couchbase/data"},
							EventingPath: "/Users/cbuser/Library/Application Support/Couchbase/var/lib/couchbase/data",
						},
					},
				},
			},
		},
		{
			name: "regression-notAllServices",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckSharedFilesystems,
						Time:   commonTestCluster.LastUpdate,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				Name: "C0",
				NodesSummary: []values.NodeSummary{
					{
						NodeUUID: "N0",
						OS:       "x86_64-unknown-linux-gnu",
						Services: []string{"kv"},
					},
				},
				LastUpdate: updateTime,
			},
			storage: &values.Storage{
				Available: values.AvailableStorage{
					DiskStorage: []values.DiskStorage{
						{
							Path: "/default",
						},
					},
				},
				NodeStorage: values.NodeStorageSet{
					HDD: []values.StorageConfig{
						{
							Path:      "/default",
							IndexPath: "/default",
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testCluster := tc.cluster
			if testCluster == nil {
				testCluster = commonTestCluster
			}
			for i, node := range testCluster.NodesSummary {
				results := sharedFileSystemsCheck(testCluster, tc.storage, node)
				wrappedResultMustMatch(tc.expected[i], results, tc.expected[i].Result.Status != values.GoodCheckerStatus,
					false, t)
			}
		})
	}
}

func TestGSISettingsChecks(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
		NodesSummary: values.NodesSummary{
			{
				NodeUUID: "N0",
			},
		},
	}
	t.Run("DefaultLogLevel", func(t *testing.T) {
		result := gsiSettingsChecks(cluster, cluster.NodesSummary[0], &values.GSISettings{
			RedistributeIndexes:    false,
			NumReplicas:            0,
			IndexerThreads:         0,
			MemorySnapshotInterval: 200,
			StableSnapshotInterval: 5000,
			MaxRollbackPoints:      2,
			LogLevel:               values.Info,
			StorageMode:            values.MemoryOptimized,
		})
		for _, res := range result {
			if res.Result.Name == values.CheckGSILogLevel {
				wrappedResultMustMatch(&values.WrappedCheckerResult{
					Cluster: cluster.UUID,
					Node:    cluster.NodesSummary[0].NodeUUID,
					Result: &values.CheckerResult{
						Name:   values.CheckGSILogLevel,
						Time:   cluster.LastUpdate,
						Value:  []byte(`"Info"`),
						Status: values.GoodCheckerStatus,
					},
				}, res, false, true, t)
			}
		}
	})
	t.Run("Changed", func(t *testing.T) {
		result := gsiSettingsChecks(cluster, cluster.NodesSummary[0], &values.GSISettings{
			RedistributeIndexes:    false,
			NumReplicas:            0,
			IndexerThreads:         0,
			MemorySnapshotInterval: 200,
			StableSnapshotInterval: 5000,
			MaxRollbackPoints:      2,
			LogLevel:               values.Debug,
			StorageMode:            values.MemoryOptimized,
		})
		for _, res := range result {
			if res.Result.Name == values.CheckGSILogLevel {
				wrappedResultMustMatch(&values.WrappedCheckerResult{
					Cluster: cluster.UUID,
					Node:    cluster.NodesSummary[0].NodeUUID,
					Result: &values.CheckerResult{
						Name:   values.CheckGSILogLevel,
						Time:   cluster.LastUpdate,
						Value:  []byte(`"Debug"`),
						Status: values.WarnCheckerStatus,
					},
				}, res, true, true, t)
			}
		}
	})
}

func TestFreeMemCheck(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
	}
	cases := []nodeTestCase{
		{
			name: "good - Under 90% usage",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckFreeMem,
						Time:   cluster.LastUpdate,
						Status: values.GoodCheckerStatus,
						Value:  []byte(`"0.00%"`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					MemTotal: 100,
					MemFree:  100,
				},
			},
		},
		{
			name: "warn - Over 90% usage",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckFreeMem,
						Time:   cluster.LastUpdate,
						Status: values.WarnCheckerStatus,
						Value:  []byte(`"99.00%"`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					MemTotal: 100,
					MemFree:  1,
				},
			},
		},
		{
			name: "good - Exactly 90% usage",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckFreeMem,
						Time:   cluster.LastUpdate,
						Status: values.GoodCheckerStatus,
						Value:  []byte(`"90.00%"`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					MemTotal: 100,
					MemFree:  10,
				},
			},
		},
	}
	runNodeTestCases(t, *cluster, cases, freeMemCheck)
}

func runNodeTestCases(t *testing.T, cluster values.CouchbaseCluster, cases []nodeTestCase,
	fn func(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error),
) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cluster.NodesSummary = tc.nodes
			results, err := fn(cluster)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(results) != len(tc.expected) {
				t.Fatalf("Expected %d results got %d", len(tc.expected), len(results))
			}

			for i, res := range results {
				wrappedResultMustMatch(tc.expected[i], res, tc.expected[i].Result.Status != values.GoodCheckerStatus,
					false, t)
			}
		})
	}
}

func TestCheckCBASJRE(t *testing.T) {
	t.Run("CB 6.5", func(t *testing.T) {
		cluster := &values.CouchbaseCluster{
			UUID:       "C0",
			Name:       "cluster",
			LastUpdate: time.Now().UTC(),
			NodesSummary: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "6.5.0",
				},
			},
		}
		createExpectedResult := func(name string, value json.RawMessage,
			status values.CheckerStatus,
		) *values.WrappedCheckerResult {
			return &values.WrappedCheckerResult{
				Cluster: cluster.UUID,
				Node:    cluster.NodesSummary[0].NodeUUID,
				Result: &values.CheckerResult{
					Name:   name,
					Value:  value,
					Status: status,
				},
			}
		}
		t.Run("Valid Vendor, Valid Version", func(t *testing.T) {
			vendors := []string{"Oracle Corporation", "Eclipse Foundation", "AdoptOpenJDK"}
			versions := []string{"1.8.0_182", "1.8.0_182", "1.8.0_182"}
			for i := range vendors {
				result, err := checkAnalyticsJRE(cluster, &cluster.NodesSummary[0], &values.AnalyticsNodeDiagnostics{
					Runtime: values.AnalyticsRuntime{
						SystemProperties: values.AnalyticsSystemProperties{
							JavaVendor:  vendors[i],
							JavaVersion: versions[i],
						},
					},
				})
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}

				// expected := createExpectedResult()
				vendorValue, err := json.Marshal(vendors[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}
				versionValue, err := json.Marshal(versions[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}
				wrappedResultMustMatch(createExpectedResult(
					values.CheckAnalyticsJRE, vendorValue, values.GoodCheckerStatus), result[0], false, true, t)
				wrappedResultMustMatch(createExpectedResult(
					values.CheckAnalyticsJRE, versionValue, values.GoodCheckerStatus), result[1], false, true, t)
			}
		})
		t.Run("CB 6.5, Invalid Vendor", func(t *testing.T) {
			vendors := []string{"LegitJVM"}
			versions := []string{"1.11.0_182"}

			for i := range vendors {
				result, err := checkAnalyticsJRE(cluster, &cluster.NodesSummary[0], &values.AnalyticsNodeDiagnostics{
					Runtime: values.AnalyticsRuntime{
						SystemProperties: values.AnalyticsSystemProperties{
							JavaVendor:  vendors[i],
							JavaVersion: versions[i],
						},
					},
				})
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}

				// expected := createExpectedResult()
				vendorValue, err := json.Marshal(vendors[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}
				versionValue, err := json.Marshal(versions[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}
				fmt.Println(result[0].Result.Remediation)
				fmt.Println(result[1].Result.Remediation)

				expectedRemediation := fmt.Sprintf("%s (Unsupported)", vendors[i])
				if result[0].Result.Remediation != expectedRemediation {
					log.Fatalf("Incorrect remediation. \nExpected %s\nFound %s", expectedRemediation, result[0].Result.Remediation)
				}

				wrappedResultMustMatch(createExpectedResult(values.CheckAnalyticsJRE, vendorValue,
					values.AlertCheckerStatus), result[0], true, true, t)
				wrappedResultMustMatch(createExpectedResult(values.CheckAnalyticsJRE, versionValue,
					values.InfoCheckerStatus), result[1], false, true, t)
			}
		})
		t.Run("CB 6.5, valid Vendor, invalid version", func(t *testing.T) {
			vendors := []string{"Oracle Corporation", "Eclipse Foundation", "AdoptOpenJDK"}
			versions := []string{"1.7.0_182", "1.7.0_182", "1.7.0_182"}

			for i := range vendors {
				result, err := checkAnalyticsJRE(cluster, &cluster.NodesSummary[0], &values.AnalyticsNodeDiagnostics{
					Runtime: values.AnalyticsRuntime{
						SystemProperties: values.AnalyticsSystemProperties{
							JavaVendor:  vendors[i],
							JavaVersion: versions[i],
						},
					},
				})
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}

				// expected := createExpectedResult()
				vendorValue, err := json.Marshal(vendors[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}
				versionValue, err := json.Marshal(versions[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}

				wrappedResultMustMatch(createExpectedResult(values.CheckAnalyticsJRE, vendorValue,
					values.GoodCheckerStatus), result[0], false, true, t)
				wrappedResultMustMatch(createExpectedResult(values.CheckAnalyticsJRE, versionValue,
					values.AlertCheckerStatus), result[1], true, true, t)
			}
		})
	})
	t.Run("CB 7.0", func(t *testing.T) {
		cluster := &values.CouchbaseCluster{
			UUID:       "C0",
			Name:       "cluster",
			LastUpdate: time.Now().UTC(),
			NodesSummary: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "7.0.0",
				},
			},
		}
		createExpectedResult := func(name string, value json.RawMessage,
			status values.CheckerStatus,
		) *values.WrappedCheckerResult {
			return &values.WrappedCheckerResult{
				Cluster: cluster.UUID,
				Node:    cluster.NodesSummary[0].NodeUUID,
				Result: &values.CheckerResult{
					Name:   name,
					Value:  value,
					Status: status,
				},
			}
		}
		t.Run("Valid Vendor, Valid Version", func(t *testing.T) {
			vendors := []string{"Oracle Corporation", "AdoptOpenJDK"}
			versions := []string{"1.11.0_182", "11.0.12"}
			for i := range vendors {
				result, err := checkAnalyticsJRE(cluster, &cluster.NodesSummary[0], &values.AnalyticsNodeDiagnostics{
					Runtime: values.AnalyticsRuntime{
						SystemProperties: values.AnalyticsSystemProperties{
							JavaVendor:  vendors[i],
							JavaVersion: versions[i],
						},
					},
				})
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}

				// expected := createExpectedResult()
				vendorValue, err := json.Marshal(vendors[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}
				versionValue, err := json.Marshal(versions[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}

				wrappedResultMustMatch(createExpectedResult(values.CheckAnalyticsJRE, vendorValue,
					values.GoodCheckerStatus), result[0], false, true, t)
				wrappedResultMustMatch(createExpectedResult(values.CheckAnalyticsJRE, versionValue,
					values.GoodCheckerStatus), result[1], false, true, t)
			}
		})
		t.Run("Invalid Vendor", func(t *testing.T) {
			vendors := []string{"LegitJVM"}
			versions := []string{"1.11.0_182"}

			for i := range vendors {
				result, err := checkAnalyticsJRE(cluster, &cluster.NodesSummary[0], &values.AnalyticsNodeDiagnostics{
					Runtime: values.AnalyticsRuntime{
						SystemProperties: values.AnalyticsSystemProperties{
							JavaVendor:  vendors[i],
							JavaVersion: versions[i],
						},
					},
				})
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}

				// expected := createExpectedResult()
				vendorValue, err := json.Marshal(vendors[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}
				versionValue, err := json.Marshal(versions[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}

				wrappedResultMustMatch(createExpectedResult(values.CheckAnalyticsJRE, vendorValue,
					values.AlertCheckerStatus), result[0], true, true, t)
				wrappedResultMustMatch(createExpectedResult(values.CheckAnalyticsJRE, versionValue,
					values.InfoCheckerStatus), result[1], false, true, t)
			}
		})
		t.Run("valid Vendor, invalid version", func(t *testing.T) {
			vendors := []string{"Oracle Corporation", "Eclipse Foundation"}
			versions := []string{"1.7.0_182", "10.7.0"}

			for i := range vendors {
				result, err := checkAnalyticsJRE(cluster, &cluster.NodesSummary[0], &values.AnalyticsNodeDiagnostics{
					Runtime: values.AnalyticsRuntime{
						SystemProperties: values.AnalyticsSystemProperties{
							JavaVendor:  vendors[i],
							JavaVersion: versions[i],
						},
					},
				})
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}

				// expected := createExpectedResult()
				vendorValue, err := json.Marshal(vendors[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}
				versionValue, err := json.Marshal(versions[i])
				if err != nil {
					t.Fatalf("Unexpected Error %v", err)
				}

				wrappedResultMustMatch(createExpectedResult(values.CheckAnalyticsJRE, vendorValue,
					values.GoodCheckerStatus), result[0], false, true, t)
				wrappedResultMustMatch(createExpectedResult(values.CheckAnalyticsJRE, versionValue,
					values.AlertCheckerStatus), result[1], true, true, t)
			}
		})
	})
}
