package status

import (
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/values"
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
						Name:   "oneServicePerNode",
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
						Name:   "oneServicePerNode",
						Status: values.GoodCheckerStatus,
						Time:   cluster.LastUpdate,
					},
					Cluster: "C0",
					Node:    "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "oneServicePerNode",
						Status: values.WarnCheckerStatus,
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
						Name:   "oneServicePerNode",
						Status: values.WarnCheckerStatus,
						Time:   cluster.LastUpdate,
						Value:  []byte(`{"node_uuid":"N0","services":["kv","backup","index"]}`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "oneServicePerNode",
						Status: values.WarnCheckerStatus,
						Value:  []byte(`{"node_uuid":"N1","services":["kv","backup"]}`),
						Time:   cluster.LastUpdate,
					},
					Cluster: "C0",
					Node:    "N1",
				},
			},
		},
	}

	runNodeTestCases(t, cluster, cases, oneServicePerNodeCheck)
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
						Name:   "supportedVersion",
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
						Name:   "supportedVersion",
						Time:   cluster.LastUpdate,
						Status: values.AlertCheckerStatus,
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
						Name:   "supportedVersion",
						Time:   cluster.LastUpdate,
						Value:  []byte(`"6.0.1-2039-enterprise"`),
						Status: values.WarnCheckerStatus,
					},
					Cluster: "uuid-0",
					Node:    "node-0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "node-0",
					Version:  "6.0.1-2039-enterprise",
				},
			},
		},
		{
			name: "unsupported-EOS",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "supportedVersion",
						Time:   cluster.LastUpdate,
						Value:  []byte(`"5.1.3-6212-enterprise"`),
						Status: values.AlertCheckerStatus,
					},
					Cluster: "uuid-0",
					Node:    "node-0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "node-0",
					Version:  "5.1.3-6212-enterprise",
				},
			},
		},
		{
			name: "mixed-supported",
			nodes: values.NodesSummary{
				{
					NodeUUID: "node-0",
					Version:  "6.0.1-2039-enterprise",
				},
				{
					NodeUUID: "node-1",
					Version:  "6.6.1-9213-enterprise",
				},
				{
					NodeUUID: "node-3",
					Version:  "5.1.3-6212-enterprise",
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "supportedVersion",
						Time:   cluster.LastUpdate,
						Value:  []byte(`"6.0.1-2039-enterprise"`),
						Status: values.WarnCheckerStatus,
					},
					Cluster: "uuid-0",
					Node:    "node-0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "supportedVersion",
						Value:  []byte(`"6.6.1-9213-enterprise"`),
						Time:   cluster.LastUpdate,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "uuid-0",
					Node:    "node-1",
				},
				{
					Result: &values.CheckerResult{
						Name:   "supportedVersion",
						Time:   cluster.LastUpdate,
						Status: values.AlertCheckerStatus,
						Value:  []byte(`"5.1.3-6212-enterprise"`),
					},
					Cluster: "uuid-0",
					Node:    "node-3",
				},
			},
		},
	}

	runNodeTestCases(t, cluster, cases, supportedVersionCheck)
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
						Name:   "unhealthyNode",
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
						Name:   "unhealthyNode",
						Status: values.GoodCheckerStatus,
						Time:   cluster.LastUpdate,
					},
					Cluster: "C0",
					Node:    "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "unhealthyNode",
						Status: values.AlertCheckerStatus,
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
						Name:   "unhealthyNode",
						Status: values.AlertCheckerStatus,
						Time:   cluster.LastUpdate,
						Value:  []byte(`{"node_uuid":"N0","status":"unhealthy","cluster_membership":"active"}`),
					},
					Cluster: "C0",
					Node:    "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "unhealthyNode",
						Status: values.AlertCheckerStatus,
						Time:   cluster.LastUpdate,
						Value:  []byte(`{"node_uuid":"N1","status":"healthy","cluster_membership":"inactive"}`),
					},
					Cluster: "C0",
					Node:    "N1",
				},
			},
		},
	}

	runNodeTestCases(t, cluster, cases, unhealthyNodesCheck)
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
						Name:   "nonGABuild",
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
					Version:  "5.5.4-4340-enterprise",
				},
			},
		},
		{
			name: "warn",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "nonGABuild",
						Time:   cluster.LastUpdate,
						Status: values.WarnCheckerStatus,
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
						Name:   "nonGABuild",
						Time:   cluster.LastUpdate,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "nonGABuild",
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
					Version:  "5.5.4-4340-enterprise",
				},
			},
		},
	}

	runNodeTestCases(t, cluster, cases, nonGABuildCheck)
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
						Name:   "nodeSwapUsage",
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
						Name:   "nodeSwapUsage",
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
						Name:   "nodeSwapUsage",
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
	}

	runNodeTestCases(t, cluster, cases, nodeSwapUsageCheck)
}

func TestCpuBucketCountCheck(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
		BucketsSummary: []values.BucketSummary{
			{
				Name: "B0",
			},
			{
				Name: "B1",
			},
		},
	}

	cases := []nodeTestCase{
		{
			name: "good",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "cpuBucketCount",
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
					CPUCount: 3,
					Services: []string{"kv"},
				},
			},
		},
		{
			name: "warning",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "cpuBucketCount",
						Time:   cluster.LastUpdate,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					CPUCount: 1,
					Services: []string{"kv"},
				},
			},
		},
		{
			name: "mixed",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "cpuBucketCount",
						Time:   cluster.LastUpdate,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "cpuBucketCount",
						Time:   cluster.LastUpdate,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Node:    "N1",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					CPUCount: 3,
					Services: []string{"kv"},
				},
				{
					NodeUUID: "N1",
					CPUCount: 1,
					Services: []string{"kv"},
				},
			},
		},
	}

	runNodeTestCases(t, cluster, cases, cpuBucketCountCheck)
}

type nodeSelfTestCase struct {
	name     string
	expected []*values.WrappedCheckerResult
	storage  *values.Storage
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
						Name:   "nodeDiskSpace",
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
						Name:   "nodeDiskSpace",
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

func runNodeTestCases(t *testing.T, cluster *values.CouchbaseCluster, cases []nodeTestCase,
	fn func(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error)) {
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
