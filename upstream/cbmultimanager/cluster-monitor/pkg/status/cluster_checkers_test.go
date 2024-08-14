// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package status

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase/mocks"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/require"
)

func TestSingleOrTwoNodeClusterCheck(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID: "uuid-0",
		Name: "cluster",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID: "0",
			},
			{
				NodeUUID: "1",
			},
		},
		LastUpdate: time.Now().UTC(),
	}

	t.Run("checker-fail", func(t *testing.T) {
		results, err := singleOrTwoNodeClusterCheck(*cluster)
		if err != nil {
			t.Fatalf("Unexpected error running checker: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 results got %d", len(results))
		}

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckSingleOrTwoNodeCluster,
				Time:   cluster.LastUpdate,
				Status: values.InfoCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], true, false, t)
	})

	t.Run("checker-pass", func(t *testing.T) {
		cluster.NodesSummary = append(cluster.NodesSummary, values.NodeSummary{NodeUUID: "2"})
		results, err := singleOrTwoNodeClusterCheck(*cluster)
		if err != nil {
			t.Fatalf("Unexpected error running checker: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 results got %d", len(results))
		}

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckSingleOrTwoNodeCluster,
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], false, false, t)
	})
}

func TestActveCluster(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID: "uuid-0",
		Name: "cluster",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:          "0",
				Status:            "healthy",
				ClusterMembership: "active",
			},
			{
				NodeUUID:          "1",
				Status:            "healthy",
				ClusterMembership: "active",
			},
		},
		LastUpdate: time.Now().UTC(),
	}

	t.Run("Checker-good", func(t *testing.T) {
		results, err := activeClusterCheck(*cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckActiveCluster,
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], false, false, t)
	})

	t.Run("Checker-warn", func(t *testing.T) {
		cluster.NodesSummary = append(cluster.NodesSummary, values.NodeSummary{
			NodeUUID:          "2",
			Status:            "unhealthy",
			Host:              "hostname2",
			ClusterMembership: "inactive",
		})
		cluster.NodesSummary = append(cluster.NodesSummary, values.NodeSummary{
			NodeUUID:          "3",
			Status:            "unhealthy",
			Host:              "hostname3",
			ClusterMembership: "inactive",
		})

		results, err := activeClusterCheck(*cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckActiveCluster,
				Time:   cluster.LastUpdate,
				Status: values.AlertCheckerStatus,
				Value:  []byte(`{"nodes":["hostname2","hostname3"]}`),
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], false, false, t)
	})
}

func TestMixedModeCheck(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID: "uuid-0",
		Name: "cluster",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID: "0",
				Version:  "7.0.0-0000-enterprise",
			},
			{
				NodeUUID: "1",
				Version:  "7.0.0-0000-enterprise",
			},
		},
		LastUpdate: time.Now().UTC(),
	}

	t.Run("checker-pass", func(t *testing.T) {
		results, err := mixedModeCheck(*cluster)
		if err != nil {
			t.Fatalf("Unexpected error running checker: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 results got %d", len(results))
		}

		out, _ := json.Marshal(map[string][]string{
			"7.0.0-0000-enterprise": {"0", "1"},
		})

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckMixedMode,
				Time:   cluster.LastUpdate,
				Value:  out,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], false, false, t)
	})

	t.Run("checker-fail", func(t *testing.T) {
		cluster.NodesSummary = append(cluster.NodesSummary,
			values.NodeSummary{NodeUUID: "a", Version: "6.6.2-0000-enterprise"})
		results, err := mixedModeCheck(*cluster)
		if err != nil {
			t.Fatalf("Unexpected error running checker: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 results got %d", len(results))
		}

		out, _ := json.Marshal(map[string][]string{
			"7.0.0-0000-enterprise": {"0", "1"},
			"6.6.2-0000-enterprise": {"a"},
		})

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckMixedMode,
				Time:   cluster.LastUpdate,
				Value:  out,
				Status: values.InfoCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], true, false, t)
	})
}

func TestDeveloperPreviewCheck(t *testing.T) {
	type ExpectedResults struct {
		CheckerResult values.WrappedCheckerResult
		Remediation   bool
	}
	testCases := []struct {
		Name     string
		Cluster  values.CouchbaseCluster
		Expected ExpectedResults
	}{
		{
			Name: "good",
			Cluster: values.CouchbaseCluster{
				UUID:             "cluster0",
				DeveloperPreview: false,
			},
			Expected: ExpectedResults{
				CheckerResult: values.WrappedCheckerResult{
					Result: &values.CheckerResult{
						Name:   values.CheckDeveloperPreview,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "cluster0",
				},
			},
		},
		{
			Name: "bad",
			Cluster: values.CouchbaseCluster{
				UUID:             "cluster0",
				DeveloperPreview: true,
			},
			Expected: ExpectedResults{
				CheckerResult: values.WrappedCheckerResult{
					Result: &values.CheckerResult{
						Name:   values.CheckDeveloperPreview,
						Status: values.AlertCheckerStatus,
					},
					Cluster: "cluster0",
				},
				Remediation: true,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := developerPreviewCheck(tc.Cluster)
			wrappedResultMustMatch(&tc.Expected.CheckerResult, result,
				tc.Expected.Remediation, true, t)
		})
	}
}

type singleRestCallTest struct {
	name             string
	expectedResult   *values.WrappedCheckerResult
	checkRemediation bool

	fnName  string
	args    []interface{}
	returns []interface{}

	poolsMeta *couchbase.PoolsMetadata
}

type singleCacheRESTDataTest struct {
	name             string
	expectedResult   *values.WrappedCheckerResult
	checkRemediation bool

	cluster values.CouchbaseCluster
}

func runSingleRestCallTests(t *testing.T, cases []singleRestCallTest,
	fn func(client couchbase.ClientIFace) *values.WrappedCheckerResult,
) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := new(mocks.ClientIFace)

			if tc.fnName != "" {
				client.On(tc.fnName, tc.args...).Return(tc.returns...)
			}

			client.On("GetBootstrap").Return(time.Time{}.UTC())

			poolsMeta := tc.poolsMeta
			if poolsMeta == nil {
				poolsMeta = &couchbase.PoolsMetadata{
					ClusterUUID:  "uuid-0",
					ClusterName:  "name-0",
					NodesSummary: values.NodesSummary{},
					ClusterInfo:  &values.ClusterInfo{},
					PoolsRaw:     []byte{},
				}
			}

			client.On("GetClusterInfo").Return(poolsMeta)

			result := fn(client)
			wrappedResultMustMatch(tc.expectedResult, result, tc.checkRemediation, true, t)
		})
	}
}

func runSingleCacheRESTDataTests(t *testing.T, cases []singleCacheRESTDataTest,
	fn func(cluster values.CouchbaseCluster) *values.WrappedCheckerResult,
) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cluster := tc.cluster

			result := fn(cluster)
			wrappedResultMustMatch(tc.expectedResult, result, tc.checkRemediation, true, t)
		})
	}
}

func TestServerQuotaCheck(t *testing.T) {
	runSingleCacheRESTDataTests(t, []singleCacheRESTDataTest{
		{
			name: "good",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckServerQuota,
					Value:  []byte(`{"quota":10.00}`),
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			cluster: values.CouchbaseCluster{
				UUID:     "uuid-0",
				PoolsRaw: []byte(`{"storageTotals": {"ram": {"quotaTotal":500, "total":5000}}}`),
			},
		},
		{
			name: "alert",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckServerQuota,
					Value:  []byte(`{"quota":100.00}`),
					Status: values.AlertCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			cluster: values.CouchbaseCluster{
				UUID:     "uuid-0",
				PoolsRaw: []byte(`{"storageTotals": {"ram": {"quotaTotal":500, "total":500}}}`),
			},
		},
		{
			name: "warn",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckServerQuota,
					Value:  []byte(`{"quota":81.00}`),
					Status: values.WarnCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			cluster: values.CouchbaseCluster{
				UUID:     "uuid-0",
				PoolsRaw: []byte(`{"storageTotals": {"ram": {"quotaTotal":405, "total":500}}}`),
			},
		},
	}, serverQuotaCheck)
}

func TestEmptyGroupCheck(t *testing.T) {
	runSingleCacheRESTDataTests(t, []singleCacheRESTDataTest{
		{
			name: "good",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckEmptyServerGroup,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			cluster: values.CouchbaseCluster{
				UUID: "uuid-0",
				CacheRESTData: values.CacheRESTData{
					ServerGroups: []values.ServerGroup{
						{
							Name: "Group1",
							Nodes: []values.GroupNodes{{
								Hostname: "0.0.0.1",
								NodeUUID: "0",
							}},
						},
					},
				},
			},
		},
		{
			name:             "bad",
			checkRemediation: true,
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckEmptyServerGroup,
					Value:  []byte(`["Group2"]`),
					Status: values.InfoCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			cluster: values.CouchbaseCluster{
				UUID: "uuid-0",
				CacheRESTData: values.CacheRESTData{
					ServerGroups: []values.ServerGroup{
						{
							Name: "Group1",
							Nodes: []values.GroupNodes{{
								Hostname: "0.0.0.0",
								NodeUUID: "0",
							}},
						},
						{
							Name:  "Group2",
							Nodes: []values.GroupNodes{},
						},
					},
				},
			},
		},
	}, emptyGroupCheck)
}

func TestGlobalAutoCompactionCheck(t *testing.T) {
	runSingleCacheRESTDataTests(t, []singleCacheRESTDataTest{
		{
			name: "good",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckGlobalAutoCompaction,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			cluster: values.CouchbaseCluster{
				UUID: "uuid-0",
				PoolsRaw: []byte(`{"autoCompactionSettings":
					{"databaseFragmentationThreshold":{"percentage":10,"size":500}}}`),
			},
		},
		{
			name: "warning",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckGlobalAutoCompaction,
					Status: values.WarnCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			cluster: values.CouchbaseCluster{
				UUID: "uuid-0",
				PoolsRaw: []byte(`{"autoCompactionSettings":
					{"databaseFragmentationThreshold":{"percentage":"undefined","size":"undefined"}}}`),
			},
		},
	}, globalAutoCompactionCheck)
}

func TestAutoFailoverChecker(t *testing.T) {
	runSingleCacheRESTDataTests(t, []singleCacheRESTDataTest{
		{
			name: "enabled",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckAutoFailoverEnabled,
					Value:  []byte(`{"enabled":true}`),
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			cluster: values.CouchbaseCluster{
				Name: "name-0",
				UUID: "uuid-0",
				CacheRESTData: values.CacheRESTData{
					AutoFailOverSettings: &values.AutoFailoverSettings{
						Enabled: true,
					},
				},
			},
		},
		{
			name: "disabled",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckAutoFailoverEnabled,
					Value:  []byte(`{"enabled":false}`),
					Status: values.WarnCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			checkRemediation: true,
			cluster: values.CouchbaseCluster{
				Name: "name-0",
				UUID: "uuid-0",
				CacheRESTData: values.CacheRESTData{
					AutoFailOverSettings: &values.AutoFailoverSettings{},
				},
			},
		},
	}, autoFailoverChecker)
}

func TestDataLossChecker(t *testing.T) {
	runSingleCacheRESTDataTests(t, []singleCacheRESTDataTest{
		{
			name: "no-data-loss",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckDataLoss,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			cluster: values.CouchbaseCluster{
				Name: "name-0",
				UUID: "uuid-0",
				CacheRESTData: values.CacheRESTData{
					UILogs: []values.UILogEntry{
						{
							Code:       0,
							Module:     "m",
							Node:       "a",
							ServerTime: "a",
							Text:       "hello",
							Type:       "ns",
						},
					},
				},
			},
		},
		{
			name: "data-loss",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckDataLoss,
					Status: values.AlertCheckerStatus,
					Value:  []byte(`{"code":0,"module":"m","node":"a","serverTime":"a","text":"lost data","type":"ns"}`),
				},
				Cluster: "uuid-0",
			},
			checkRemediation: true,
			cluster: values.CouchbaseCluster{
				Name: "name-0",
				UUID: "uuid-0",
				CacheRESTData: values.CacheRESTData{
					UILogs: []values.UILogEntry{
						{
							Code:       0,
							Module:     "m",
							Node:       "a",
							ServerTime: "a",
							Text:       "lost data",
							Type:       "ns",
						},
					},
				},
			},
		},
	}, dataLossChecker)
}

func TestBackupLocationCheck(t *testing.T) {
	c := time.Now()
	runSingleRestCallTests(t, []singleRestCallTest{
		{
			name: "good-empty-values",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckBackupLocation,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			fnName: "GetMetric",
			args: []interface{}{
				c.AddDate(0, 0, -3).Format(time.RFC3339), c.Format(time.RFC3339),
				"backup_location_check", "10m",
			},
			returns: []interface{}{&couchbase.Metric{
				Values: []couchbase.MetricVal{},
				Name:   "backup_location_check",
			}, nil},
		},
		{
			name: "good-populated-values",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckBackupLocation,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			fnName: "GetMetric",
			args: []interface{}{
				c.AddDate(0, 0, -3).Format(time.RFC3339), c.Format(time.RFC3339),
				"backup_location_check", "10m",
			},
			returns: []interface{}{&couchbase.Metric{
				Values: []couchbase.MetricVal{
					{
						Timestamp: time.Unix(1621414560, 0),
						Value:     "0",
					},
					{
						Timestamp: time.Unix(1621414561, 0),
						Value:     "0",
					},
				},
				Name: "backup_location_check",
			}, nil},
		},
		{
			name: "bad",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckBackupLocation,
					Status: values.WarnCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			fnName: "GetMetric",
			args: []interface{}{
				c.AddDate(0, 0, -3).Format(time.RFC3339), c.Format(time.RFC3339),
				"backup_location_check", "10m",
			},
			returns: []interface{}{&couchbase.Metric{
				Values: []couchbase.MetricVal{
					{
						Timestamp: time.Unix(1621414560, 0),
						Value:     "0",
					},
					{
						Timestamp: time.Unix(1621414561, 0),
						Value:     "1",
					},
					{
						Timestamp: time.Unix(1621414562, 0),
						Value:     "2",
					},
				},
				Name: "backup_location_check",
			}, nil},
		},
	}, backupLocationCheck)
}

func TestBackupOrphanedCheck(t *testing.T) {
	c := time.Now()
	runSingleRestCallTests(t, []singleRestCallTest{
		{
			name: "good-empty-values",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckBackupTaskOrphaned,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			fnName: "GetMetric",
			args: []interface{}{
				c.AddDate(0, 0, -3).Format(time.RFC3339), c.Format(time.RFC3339),
				"backup_task_orphaned", "10m",
			},
			returns: []interface{}{&couchbase.Metric{
				Values: []couchbase.MetricVal{},
				Name:   "backup_task_orphaned",
			}, nil},
		},
		{
			name: "good-populated-values",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckBackupTaskOrphaned,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			fnName: "GetMetric",
			args: []interface{}{
				c.AddDate(0, 0, -3).Format(time.RFC3339), c.Format(time.RFC3339),
				"backup_task_orphaned", "10m",
			},
			returns: []interface{}{&couchbase.Metric{
				Values: []couchbase.MetricVal{
					{
						Timestamp: time.Unix(1621414560, 0),
						Value:     "0",
					},
					{
						Timestamp: time.Unix(1621414561, 0),
						Value:     "0",
					},
				},
				Name: "backup_task_orphaned",
				Repo: "repository",
			}, nil},
		},
		{
			name: "bad",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckBackupTaskOrphaned,
					Status: values.WarnCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			fnName: "GetMetric",
			args: []interface{}{
				c.AddDate(0, 0, -3).Format(time.RFC3339), c.Format(time.RFC3339),
				"backup_task_orphaned", "10m",
			},
			returns: []interface{}{&couchbase.Metric{
				Values: []couchbase.MetricVal{
					{
						Timestamp: time.Unix(1621414560, 0),
						Value:     "0",
					},
					{
						Timestamp: time.Unix(1621414561, 0),
						Value:     "1",
					},
				},
				Name: "backup_task_orphaned",
				Repo: "repository",
			}, nil},
		},
	}, backupTaskOrphaned)
}

func TestAsymmetricClusterCheck(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID: "uuid-0",
		Name: "cluster",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID:  "0",
				MemTotal:  1000,
				SwapTotal: 100,
				Services:  []string{"kv", "index"},
			},
		},
		LastUpdate: time.Now().UTC(),
	}

	t.Run("2-nodes-no-services", func(t *testing.T) {
		cluster.NodesSummary = append(cluster.NodesSummary, values.NodeSummary{
			NodeUUID:  "1",
			Services:  []string{"query"},
			MemTotal:  1000,
			SwapTotal: 100,
		})
		results, err := asymmetricalClusterCheck(*cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckAsymmetricalCluster,
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], false, false, t)
	})

	t.Run("2-nodes-same-services-symmetrical", func(t *testing.T) {
		cluster.NodesSummary = append(cluster.NodesSummary, values.NodeSummary{
			NodeUUID:  "1",
			Services:  []string{"kv", "query"},
			MemTotal:  1000,
			SwapTotal: 100,
		})
		results, err := asymmetricalClusterCheck(*cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckAsymmetricalCluster,
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], false, false, t)
	})

	t.Run("2-nodes-same-services-asymmetrical", func(t *testing.T) {
		cluster := &values.CouchbaseCluster{
			UUID: "uuid-0",
			Name: "cluster",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID:  "0",
					MemTotal:  1000,
					SwapTotal: 100,
					Services:  []string{"kv", "index"},
				},
				{
					NodeUUID:  "0",
					MemTotal:  2000,
					SwapTotal: 10,
					Services:  []string{"kv", "index"},
				},
			},
			LastUpdate: time.Now().UTC(),
		}
		results, err := asymmetricalClusterCheck(*cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckAsymmetricalCluster,
				Time:   cluster.LastUpdate,
				Status: values.InfoCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], true, false, t)
	})

	t.Run("3-nodes-same-services-mixed", func(t *testing.T) {
		cluster.NodesSummary = append(cluster.NodesSummary, values.NodeSummary{
			NodeUUID:  "1",
			Services:  []string{"kv"},
			MemTotal:  100,
			SwapTotal: 500,
		})
		results, err := asymmetricalClusterCheck(*cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckAsymmetricalCluster,
				Time:   cluster.LastUpdate,
				Status: values.InfoCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], true, false, t)
	})
}

func TestDuplicateNodeUUIDs(t *testing.T) {
	updateTime := time.Now().UTC()

	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: updateTime,
	}

	dupValue := make([]string, 0)
	dupValue = append(dupValue, "N0")
	JSONdupValue, _ := json.Marshal(dupValue)

	cases := []nodeTestCase{
		{
			name: "good - no duplicates",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckDuplicateNodeUUID,
						Time:   updateTime,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
				},
				{
					NodeUUID: "N1",
				},
			},
		},
		{
			name: "alert - duplicates",
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:        values.CheckDuplicateNodeUUID,
						Time:        updateTime,
						Status:      values.AlertCheckerStatus,
						Remediation: "",
						Value:       JSONdupValue,
					},
					Cluster: "C0",
				},
			},
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
				},
				{
					NodeUUID: "N0",
				},
				{
					NodeUUID: "N1",
				},
			},
		},
	}

	for _, tCase := range cases {
		t.Run(tCase.name, func(t *testing.T) {
			cluster.NodesSummary = tCase.nodes
			results, err := checkDuplicateNodeUUIDs(*cluster)
			if err != nil {
				t.Fatalf("Unexpected error running checker: %v", err)
			}

			wrappedResultMustMatch(tCase.expected[0], results[0], false, false, t)
		})
	}
}

func TestTooManyFTSIndexReplicas(t *testing.T) {
	type testFTS struct {
		Name            string
		Cluster         values.CouchbaseCluster
		ExpectedResults values.WrappedCheckerResult
		Remediation     bool
	}

	extremeReplicasValue := []byte(`["index-1","index-2"]`)
	badReplicasValue := []byte(`["index-2"]`)

	testCases := []testFTS{
		{
			Name: "good-empty",
			Cluster: values.CouchbaseCluster{
				UUID:         "cluster0",
				NodesSummary: []values.NodeSummary{},
				CacheRESTData: values.CacheRESTData{
					FTSIndexStatus: values.FTSIndexStatus{},
				},
			},
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckTooManySearchReplicas,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "cluster0",
			},
		},
		{
			Name: "good - no replicas",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Version: "6.5.0-0000-enterprise", Services: []string{"fts"}},
				},
				CacheRESTData: values.CacheRESTData{
					FTSIndexStatus: values.FTSIndexStatus{
						Status: "ok",
						IndexDefs: values.FTSIndexDefs{
							IndexDefs: map[string]values.SingleFTSIndex{
								"index-1": {
									Name:       "index-1",
									SourceName: "bucket-1",
									PlanParameters: values.FTSPlanParams{
										NumReplicas: 0,
									},
								},
							},
						},
					},
				},
			},
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckTooManySearchReplicas,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "cluster0",
			},
		},
		{
			Name: "good - replicas",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Version: "6.5.0-0000-enterprise", Services: []string{"fts"}},
					{Version: "6.5.0-0000-enterprise", Services: []string{"fts"}},
					{Version: "6.5.0-0000-enterprise", Services: []string{"data"}},
				},
				CacheRESTData: values.CacheRESTData{
					FTSIndexStatus: values.FTSIndexStatus{
						Status: "ok",
						IndexDefs: values.FTSIndexDefs{
							IndexDefs: map[string]values.SingleFTSIndex{
								"index-1": {
									Name:       "index-1",
									SourceName: "bucket-1",
									PlanParameters: values.FTSPlanParams{
										NumReplicas: 1,
									},
								},
							},
						},
					},
				},
			},
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckTooManySearchReplicas,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "cluster0",
			},
		},
		{
			Name: "bad - tooManyReplicas",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Version: "6.5.0-0000-enterprise", Host: "N1", Services: []string{"index"}},
					{Version: "6.5.0-0000-enterprise", Host: "N1", Services: []string{"data"}},
					{Version: "6.5.0-0000-enterprise", Host: "N2", Services: []string{"fts"}},
					{Version: "6.5.0-0000-enterprise", Host: "N3", Services: []string{"data"}},
				},
				CacheRESTData: values.CacheRESTData{
					FTSIndexStatus: values.FTSIndexStatus{
						Status: "ok",
						IndexDefs: values.FTSIndexDefs{
							IndexDefs: map[string]values.SingleFTSIndex{
								"index-2": {
									Name:       "index-2",
									SourceName: "bucket-2",
									PlanParameters: values.FTSPlanParams{
										NumReplicas: 1,
									},
								},
							},
						},
					},
				},
			},
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckTooManySearchReplicas,
					Status: values.AlertCheckerStatus,
					Value:  badReplicasValue,
				},
				Cluster: "cluster0",
			},
			Remediation: true,
		},
		{
			Name: "extreme - too many replicas",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Version: "6.5.0-0000-enterprise", Host: "N1", Services: []string{"index"}},
					{Version: "6.5.0-0000-enterprise", Host: "N1", Services: []string{"data"}},
					{Version: "6.5.0-0000-enterprise", Host: "N2", Services: []string{"fts"}},
					{Version: "6.5.0-0000-enterprise", Host: "N3", Services: []string{"data"}},
					{Version: "6.5.0-0000-enterprise", Host: "N4", Services: []string{"data"}},
				},
				CacheRESTData: values.CacheRESTData{
					FTSIndexStatus: values.FTSIndexStatus{
						Status: "ok",
						IndexDefs: values.FTSIndexDefs{
							IndexDefs: map[string]values.SingleFTSIndex{
								"index-1": {
									Name:       "index-1",
									SourceName: "bucket-1",
									PlanParameters: values.FTSPlanParams{
										NumReplicas: 3,
									},
								},
								"index-2": {
									Name:       "index-2",
									SourceName: "bucket-2",
									PlanParameters: values.FTSPlanParams{
										NumReplicas: 3,
									},
								},
							},
						},
					},
				},
			},
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckTooManySearchReplicas,
					Status: values.AlertCheckerStatus,
					Value:  extremeReplicasValue,
				},
				Cluster: "cluster0",
			},
			Remediation: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			results := checkTooManySearchReplicas(tc.Cluster)
			wrappedResultMustMatch(&tc.ExpectedResults, results, tc.Remediation, true, t)
		})
	}
}

func TestIndexesChecks(t *testing.T) {
	testCases := []struct {
		Name            string
		Cluster         values.CouchbaseCluster
		ExpectedResults []struct {
			values.WrappedCheckerResult
			Remediation bool
		}
	}{
		{
			Name: "empty",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				CacheRESTData: values.CacheRESTData{
					IndexStatus: []*values.IndexStatus{},
				},
			},
			ExpectedResults: []struct {
				values.WrappedCheckerResult
				Remediation bool
			}{
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckIndexWithNoRedundancy,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckBadRedundantIndex,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckTooManyIndexReplicas,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckMissingIndexPartitions,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
			},
		},
		{
			Name: "good",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Services: []string{"index"}},
					{Services: []string{"index"}},
				},
				CacheRESTData: values.CacheRESTData{
					IndexStatus: []*values.IndexStatus{
						{
							IndexName:    "test",
							Name:         "test",
							NumReplica:   1,
							PartitionMap: map[string][]int{"Node1": {1}, "Node2": {2}},
							Definition:   "CREATE INDEX `test` ON `test` (`test`) WITH {\"num_partition\":2}",
						},
						{
							IndexName:    "test",
							Name:         "test (replica 1)",
							NumReplica:   1,
							PartitionMap: map[string][]int{"Node1": {1}, "Node2": {2}},
							Definition:   "CREATE INDEX `test` ON `test` (`test`) WITH {\"num_partition\":2}",
						},
					},
				},
			},
			ExpectedResults: []struct {
				values.WrappedCheckerResult
				Remediation bool
			}{
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckIndexWithNoRedundancy,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckBadRedundantIndex,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckTooManyIndexReplicas,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckMissingIndexPartitions,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
			},
		},
		{
			Name: "noRedundancy",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Services: []string{"index"}},
					{Services: []string{"index"}},
				},
				CacheRESTData: values.CacheRESTData{
					IndexStatus: []*values.IndexStatus{
						{
							IndexName:    "test",
							Name:         "test",
							PartitionMap: map[string][]int{"Node1": {1}},
							Definition:   "CREATE INDEX `test` ON `test` (`test`)",
						},
					},
				},
			},
			ExpectedResults: []struct {
				values.WrappedCheckerResult
				Remediation bool
			}{
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckIndexWithNoRedundancy,
							Status: values.WarnCheckerStatus,
							Value:  []byte(`["test"]`),
						},
						Cluster: "cluster0",
					},
					Remediation: true,
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckBadRedundantIndex,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckTooManyIndexReplicas,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckMissingIndexPartitions,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
			},
		},
		{
			Name: "badRedundancy",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Host: "N1", Services: []string{"index"}},
					{Host: "N2", Services: []string{"index"}},
				},
				CacheRESTData: values.CacheRESTData{
					IndexStatus: []*values.IndexStatus{
						{
							IndexName:  "test",
							Name:       "test",
							Definition: "CREATE INDEX `test` ON `test` (`test`)",
							Hosts:      []string{"N1"},
						},
						{
							IndexName:  "test2",
							Name:       "test2",
							Definition: "CREATE INDEX `test2` ON `test` (`test`)",
							Hosts:      []string{"N1"},
						},
					},
				},
			},
			ExpectedResults: []struct {
				values.WrappedCheckerResult
				Remediation bool
			}{
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckIndexWithNoRedundancy,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckBadRedundantIndex,
							Status: values.WarnCheckerStatus,
							Value:  []byte(`["test","test2"]`),
						},
						Cluster: "cluster0",
					},
					Remediation: true,
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckTooManyIndexReplicas,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckMissingIndexPartitions,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
			},
		},
		{
			Name: "tooManyReplicas",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Host: "N1", Services: []string{"index"}},
				},
				CacheRESTData: values.CacheRESTData{
					IndexStatus: []*values.IndexStatus{
						{
							IndexName:  "test",
							Name:       "test",
							Definition: "CREATE INDEX `test` ON `test` (`test`) WITH {\"num_replicas\":1}",
							Hosts:      []string{"N1"},
							NumReplica: 1,
						},
						{
							IndexName:  "test",
							Name:       "test (replica 1)",
							Definition: "CREATE INDEX `test` ON `test` (`test`) WITH {\"num_replicas\":1}",
							Hosts:      []string{"N2"},
							NumReplica: 1,
						},
					},
				},
			},
			ExpectedResults: []struct {
				values.WrappedCheckerResult
				Remediation bool
			}{
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckIndexWithNoRedundancy,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckBadRedundantIndex,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckTooManyIndexReplicas,
							Status: values.WarnCheckerStatus,
							Value:  []byte(`["test"]`),
						},
						Cluster: "cluster0",
					},
					Remediation: true,
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckMissingIndexPartitions,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
			},
		},
		{
			Name: "missingIndexPartitions",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Host: "N1", Services: []string{"index"}},
				},
				CacheRESTData: values.CacheRESTData{
					IndexStatus: []*values.IndexStatus{
						{
							IndexName:    "test",
							Name:         "test",
							PartitionMap: map[string][]int{"Node1": {1, 3}, "Node2": {2}},
							Definition:   "CREATE INDEX `test` ON `test` (`test`) WITH {\"num_partition\":4}",
							NumReplica:   1,
						},
						{
							IndexName:    "test",
							Name:         "test (replica 1)",
							PartitionMap: map[string][]int{"Node1": {1, 3}, "Node2": {2, 4}},
							Definition:   "CREATE INDEX `test` ON `test` (`test`) WITH {\"num_partition\":4}",
							NumReplica:   1,
						},
					},
				},
			},
			ExpectedResults: []struct {
				values.WrappedCheckerResult
				Remediation bool
			}{
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckIndexWithNoRedundancy,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckBadRedundantIndex,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckTooManyIndexReplicas,
							Status: values.GoodCheckerStatus,
						},
						Cluster: "cluster0",
					},
				},
				{
					WrappedCheckerResult: values.WrappedCheckerResult{
						Result: &values.CheckerResult{
							Name:   values.CheckMissingIndexPartitions,
							Status: values.AlertCheckerStatus,
							Value:  []byte(`["test"]`),
						},
						Cluster: "cluster0",
					},
					Remediation: true,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			results, err := indexesChecks(tc.Cluster)
			require.NoError(t, err)
			if len(results) != len(tc.ExpectedResults) {
				t.Fatalf("incorrect number of results: expected %d, got %d", len(tc.ExpectedResults), len(results))
			}
			for i, result := range results {
				wrappedResultMustMatch(&tc.ExpectedResults[i].WrappedCheckerResult, result,
					tc.ExpectedResults[i].Remediation, true, t)
			}
		})
	}
}

func TestImbalancedIndexPartitionCheck(t *testing.T) {
	type ExpectedResults struct {
		CheckerResult values.WrappedCheckerResult
		Remediation   bool
	}
	testCases := []struct {
		Name     string
		Cluster  values.CouchbaseCluster
		Expected ExpectedResults
	}{
		{
			Name: "empty",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				CacheRESTData: values.CacheRESTData{
					IndexStorageStats: []*values.IndexStatsStorage{},
				},
			},
			Expected: ExpectedResults{
				CheckerResult: values.WrappedCheckerResult{
					Result: &values.CheckerResult{
						Name:   values.CheckImbalancedIndexPartitions,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "cluster0",
				},
			},
		},
		{
			Name: "goodNoIndexNodes",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Services: []string{"data"}},
				},
				CacheRESTData: values.CacheRESTData{
					IndexStorageStats: []*values.IndexStatsStorage{},
				},
			},
			Expected: ExpectedResults{
				CheckerResult: values.WrappedCheckerResult{
					Result: &values.CheckerResult{
						Name:   values.CheckImbalancedIndexPartitions,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "cluster0",
				},
			},
		},
		{
			Name: "goodNoPartitions",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Services: []string{"index"}},
				},
				CacheRESTData: values.CacheRESTData{
					IndexStorageStats: []*values.IndexStatsStorage{
						{
							Name:        "testBucket:testIndex",
							PartitionID: 0,
							Stats: values.GSIMainStore{
								GSIStore: values.GSIStore{
									IndexMemory: 100,
								},
							},
						},
					},
				},
			},
			Expected: ExpectedResults{
				CheckerResult: values.WrappedCheckerResult{
					Result: &values.CheckerResult{
						Name:   values.CheckImbalancedIndexPartitions,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "cluster0",
				},
			},
		},
		{
			Name: "goodPartitions",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Services: []string{"index"}},
					{Services: []string{"index"}},
				},
				CacheRESTData: values.CacheRESTData{
					IndexStorageStats: []*values.IndexStatsStorage{
						{
							Name:        "testBucket:testIndex",
							PartitionID: 0,
							Stats: values.GSIMainStore{
								GSIStore: values.GSIStore{
									IndexMemory: 100,
								},
							},
						},
						{
							Name:        "testBucket:testIndex",
							PartitionID: 1,
							Stats: values.GSIMainStore{
								GSIStore: values.GSIStore{
									IndexMemory: 100,
								},
							},
						},
					},
				},
			},
			Expected: ExpectedResults{
				CheckerResult: values.WrappedCheckerResult{
					Result: &values.CheckerResult{
						Name:   values.CheckImbalancedIndexPartitions,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "cluster0",
				},
			},
		},
		{
			Name: "WarnImbalancedPartitions",
			Cluster: values.CouchbaseCluster{
				UUID: "cluster0",
				NodesSummary: []values.NodeSummary{
					{Services: []string{"index"}},
					{Services: []string{"index"}},
				},
				CacheRESTData: values.CacheRESTData{
					IndexStorageStats: []*values.IndexStatsStorage{
						{
							Name:        "testBucket:testIndex",
							PartitionID: 0,
							Stats: values.GSIMainStore{
								GSIStore: values.GSIStore{
									IndexMemory: 100,
								},
							},
						},
						{
							Name:        "testBucket:testIndex",
							PartitionID: 1,
							Stats: values.GSIMainStore{
								GSIStore: values.GSIStore{
									IndexMemory: 0,
								},
							},
						},
					},
				},
			},
			Expected: ExpectedResults{
				CheckerResult: values.WrappedCheckerResult{
					Result: &values.CheckerResult{
						Name:   values.CheckImbalancedIndexPartitions,
						Status: values.WarnCheckerStatus,
						Value:  []byte(`{"testBucket:testIndex":{"0":100,"1":0}}`),
					},
					Cluster: "cluster0",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := imbalancedIndexPartitionsCheck(tc.Cluster)
			wrappedResultMustMatch(&tc.Expected.CheckerResult, result,
				tc.Expected.Remediation, true, t)
		})
	}
}

// wrappedResultMustMatch gives a bit more information about failures than a reflect.DeepEquals at the top level would.
func wrappedResultMustMatch(expected, value *values.WrappedCheckerResult, shouldHaveRemediation, ignoreTime bool,
	t *testing.T,
) {
	// we do not care about the exact error just make sure that either both nil or non-nil
	if (expected.Error == nil) != (value.Error == nil) {
		t.Fatalf("Errors do not match %v vs %v", expected.Error, value.Error)
	}

	if expected.Error != nil {
		return
	}

	// the remediation strings are likely to change over time so just check that is not empty and then set to ""
	if shouldHaveRemediation && (value.Result == nil || len(value.Result.Remediation) == 0) {
		t.Fatal("Expected remediation in the results")
	}

	if value.Result != nil {
		value.Result.Remediation = ""
	}

	if ignoreTime && value.Result != nil && expected.Result != nil {
		value.Result.Time = expected.Result.Time
	}

	// Special-case Value, since it's a json.RawMessage which will get printed as a []byte otherwise - makes debugging
	// easier
	if !reflect.DeepEqual(expected.Result.Value, value.Result.Value) {
		t.Fatalf("Inner values do not match\n%s\n%s", string(expected.Result.Value), string(value.Result.Value))
	}

	if !reflect.DeepEqual(expected.Result, value.Result) {
		t.Fatalf("Inner results do not match\n%+v\n%+v", expected.Result, value.Result)
	}

	if expected.Cluster != value.Cluster {
		t.Fatalf("Clusters do not match %s vs %s", expected.Cluster, value.Cluster)
	}

	if expected.Bucket != value.Bucket {
		t.Fatalf("Buckets do not match %s vs %s", expected.Bucket, value.Bucket)
	}

	if expected.Node != value.Node {
		t.Fatalf("Nodes do not match %s vs %s", expected.Node, value.Node)
	}

	if expected.LogFile != value.LogFile {
		t.Fatalf("LogFiles do not match %s vs %s", expected.LogFile, value.LogFile)
	}
}

func generateBucketsSummaryHelper(number int) values.BucketsSummary {
	var bucketsSummary values.BucketsSummary
	for i := 0; i < number; i++ {
		bucketsSummary = append(bucketsSummary, values.BucketSummary{
			Name: "B" + strconv.Itoa(i),
		})
	}
	return bucketsSummary
}

func TestBucketCountChecks(t *testing.T) {
	type testBucketCount struct {
		Name            string
		Cluster         values.CouchbaseCluster
		ExpectedResults values.WrappedCheckerResult
		Remediation     bool
	}

	defaultBuckets := generateBucketsSummaryHelper(2)

	bucketsFor10 := generateBucketsSummaryHelper(10)

	bucketsFor30 := generateBucketsSummaryHelper(32)

	cluster := &values.CouchbaseCluster{
		LastUpdate: time.Now().UTC(),
	}

	cases := []testBucketCount{
		{
			Name: "good",
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.BucketCountChecks,
					Time:   cluster.LastUpdate,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "C0",
			},
			Cluster: values.CouchbaseCluster{
				UUID: "C0",
				Name: "cluster",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						CPUCount: 3,
						Services: []string{"kv"},
					},
				},
				BucketsSummary: defaultBuckets,
			},
			Remediation: false,
		},
		{
			Name: "warning",
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.BucketCountChecks,
					Time:   cluster.LastUpdate,
					Status: values.InfoCheckerStatus,
				},
				Cluster: "C0",
			},
			Cluster: values.CouchbaseCluster{
				UUID: "C0",
				Name: "cluster",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						CPUCount: 1,
						Services: []string{"kv"},
					},
				},
				BucketsSummary: defaultBuckets,
			},
			Remediation: true,
		},
		{
			Name: "mixed",
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.BucketCountChecks,
					Time:   cluster.LastUpdate,
					Status: values.InfoCheckerStatus,
				},
				Cluster: "C0",
			},
			Cluster: values.CouchbaseCluster{
				UUID: "C0",
				Name: "cluster",
				NodesSummary: values.NodesSummary{
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
				BucketsSummary: defaultBuckets,
			},
			Remediation: true,
		},
		{
			Name: "0.4 or below cores per bucket check in 6.x - Good",
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.BucketCountChecks,
					Time:   cluster.LastUpdate,
					Status: values.GoodCheckerStatus,
				},
				Cluster: "C0",
			},
			Cluster: values.CouchbaseCluster{
				UUID: "C0",
				Name: "cluster",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						CPUCount: 10,
						Version:  "6.0.0",
						Services: []string{"kv"},
					},
				},
				BucketsSummary: bucketsFor10,
			},
			Remediation: false,
		},
		{
			Name: "0.4 cores per bucket check in 6.x - Bad",
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.BucketCountChecks,
					Time:   cluster.LastUpdate,
					Status: values.WarnCheckerStatus,
				},
				Cluster: "C0",
			},
			Cluster: values.CouchbaseCluster{
				UUID: "C0",
				Name: "cluster",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						CPUCount: 3,
						Version:  "6.0.0",
						Services: []string{"kv"},
					},
				},
				BucketsSummary: bucketsFor10,
			},
			Remediation: true,
		},
		{
			Name: "0.2 cores per bucket check in 7.x - good",
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.BucketCountChecks,
					Time:   cluster.LastUpdate,
					Status: values.InfoCheckerStatus,
				},
				Cluster: "C0",
			},
			Cluster: values.CouchbaseCluster{
				UUID: "C0",
				Name: "cluster",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						CPUCount: 3,
						Version:  "7.0.0",
						Services: []string{"kv"},
					},
				},
				BucketsSummary: bucketsFor10,
			},
			Remediation: true,
		},
		{
			Name: "0.2 cores per bucket check in 7.x - bad",
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.BucketCountChecks,
					Time:   cluster.LastUpdate,
					Status: values.WarnCheckerStatus,
				},
				Cluster: "C0",
			},
			Cluster: values.CouchbaseCluster{
				UUID: "C0",
				Name: "cluster",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						CPUCount: 2,
						Version:  "7.0.0",
						Services: []string{"kv"},
					},
				},
				BucketsSummary: bucketsFor10,
			},
			Remediation: true,
		},
		{
			Name: "greater than 30 buckets",
			ExpectedResults: values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.BucketCountChecks,
					Time:   cluster.LastUpdate,
					Status: values.WarnCheckerStatus,
				},
				Cluster: "C0",
			},
			Cluster: values.CouchbaseCluster{
				UUID: "C0",
				Name: "cluster",
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						CPUCount: 40,
						Version:  "7.0.0",
						Services: []string{"kv"},
					},
				},
				BucketsSummary: bucketsFor30,
			},
			Remediation: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := bucketCountChecks(tc.Cluster)
			wrappedResultMustMatch(&tc.ExpectedResults, result,
				tc.Remediation, true, t)
		})
	}
}
