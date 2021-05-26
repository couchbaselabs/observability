package status

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/couchbase"
	"github.com/couchbaselabs/cbmultimanager/couchbase/mocks"
	"github.com/couchbaselabs/cbmultimanager/values"

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
		results, err := singleOrTwoNodeClusterCheck(cluster)
		if err != nil {
			t.Fatalf("Unexpected error running checker: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 results got %d", len(results))
		}

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "singleOrTwoNodeCluster",
				Time:   cluster.LastUpdate,
				Status: values.WarnCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], true, false, t)
	})

	t.Run("checker-pass", func(t *testing.T) {
		cluster.NodesSummary = append(cluster.NodesSummary, values.NodeSummary{NodeUUID: "2"})
		results, err := singleOrTwoNodeClusterCheck(cluster)
		if err != nil {
			t.Fatalf("Unexpected error running checker: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 results got %d", len(results))
		}

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "singleOrTwoNodeCluster",
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
		results, err := activeClusterCheck(cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "activeCluster",
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

		results, err := activeClusterCheck(cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "activeCluster",
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
		results, err := mixedModeCheck(cluster)
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
				Name:   "mixedMode",
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
		results, err := mixedModeCheck(cluster)
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
				Name:   "mixedMode",
				Time:   cluster.LastUpdate,
				Value:  out,
				Status: values.WarnCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], true, false, t)
	})
}

func TestGlobalAuthCompactionCheck(t *testing.T) {
	client := &couchbase.Client{
		ClusterInfo: &couchbase.PoolsMetadata{
			ClusterUUID: "uuid-0",
			ClusterName: "name",
			PoolsRaw: []byte(`{"autoCompactionSettings":
				{"databaseFragmentationThreshold":{"percentage":10,"size":500}}}`),
		},
		BootstrapTime: time.Now().UTC(),
	}

	t.Run("checker-pass", func(t *testing.T) {
		result := globalAutoCompactionCheck(client)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "globalAutoCompaction",
				Time:   client.BootstrapTime,
				Status: values.GoodCheckerStatus,
			},
			Cluster: "uuid-0",
		}

		wrappedResultMustMatch(expectedResult, result, false, false, t)
	})

	t.Run("checker-alert", func(t *testing.T) {
		client.ClusterInfo.PoolsRaw = []byte(`{"autoCompactionSettings":
			{"databaseFragmentationThreshold":{"percentage":"undefined","size":"undefined"}}}`)
		result := globalAutoCompactionCheck(client)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "globalAutoCompaction",
				Time:   client.BootstrapTime,
				Status: values.AlertCheckerStatus,
			},
			Cluster: "uuid-0",
		}

		wrappedResultMustMatch(expectedResult, result, true, false, t)
	})
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

func runSingleRestCallTests(t *testing.T, cases []singleRestCallTest,
	fn func(client couchbase.ClientIFace) *values.WrappedCheckerResult) {
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

func TestServerQuotaCheck(t *testing.T) {
	runSingleRestCallTests(t, []singleRestCallTest{
		{
			name: "good",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "serverQuota",
					Value:  []byte(`{"quota":10.00}`),
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			poolsMeta: &couchbase.PoolsMetadata{
				ClusterUUID: "uuid-0",
				PoolsRaw:    []byte(`{"storageTotals": {"ram": {"quotaTotal":500, "total":5000}}}`),
			},
		},
		{
			name: "alert",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "serverQuota",
					Value:  []byte(`{"quota":100.00}`),
					Status: values.AlertCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			poolsMeta: &couchbase.PoolsMetadata{
				ClusterUUID: "uuid-0",
				PoolsRaw:    []byte(`{"storageTotals": {"ram": {"quotaTotal":500, "total":500}}}`),
			},
		},
		{
			name: "warn",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "serverQuota",
					Value:  []byte(`{"quota":81.00}`),
					Status: values.WarnCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			poolsMeta: &couchbase.PoolsMetadata{
				ClusterUUID: "uuid-0",
				PoolsRaw:    []byte(`{"storageTotals": {"ram": {"quotaTotal":405, "total":500}}}`),
			},
		},
	}, serverQuotaCheck)
}

func TestGlobalAutoCompactionCheck(t *testing.T) {
	runSingleRestCallTests(t, []singleRestCallTest{
		{
			name: "good",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "globalAutoCompaction",
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			poolsMeta: &couchbase.PoolsMetadata{
				ClusterUUID: "uuid-0",
				PoolsRaw: []byte(`{"autoCompactionSettings":
					{"databaseFragmentationThreshold":{"percentage":10,"size":500}}}`),
			},
		},
		{
			name: "alert",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "globalAutoCompaction",
					Status: values.AlertCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			poolsMeta: &couchbase.PoolsMetadata{
				ClusterUUID: "uuid-0",
				PoolsRaw: []byte(`{"autoCompactionSettings":
					{"databaseFragmentationThreshold":{"percentage":"undefined","size":"undefined"}}}`),
			},
		},
	}, globalAutoCompactionCheck)
}

func TestAutoFailoverChecker(t *testing.T) {
	runSingleRestCallTests(t, []singleRestCallTest{
		{
			name: "error",
			expectedResult: &values.WrappedCheckerResult{
				Error: values.ErrNotFound,
			},
			fnName:  "GetAutoFailOverSettings",
			returns: []interface{}{nil, values.ErrNotFound},
		},
		{
			name: "enabled",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "autoFailoverEnabled",
					Value:  []byte(`{"enabled":true}`),
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			fnName:  "GetAutoFailOverSettings",
			returns: []interface{}{&couchbase.AutoFailoverSettings{Enabled: true}, nil},
		},
		{
			name: "disabled",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "autoFailoverEnabled",
					Value:  []byte(`{"enabled":false}`),
					Status: values.WarnCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			checkRemediation: true,
			fnName:           "GetAutoFailOverSettings",
			returns:          []interface{}{&couchbase.AutoFailoverSettings{}, nil},
		},
	}, autoFailoverChecker)
}

func TestDataLossChecker(t *testing.T) {
	runSingleRestCallTests(t, []singleRestCallTest{
		{
			name: "error",
			expectedResult: &values.WrappedCheckerResult{
				Error: values.ErrNotFound,
			},
			fnName:  "GetUILogs",
			returns: []interface{}{[]couchbase.UILogEntry{}, values.ErrNotFound},
		},
		{
			name: "no-data-loss",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "dataLoss",
					Status: values.GoodCheckerStatus,
				},
				Cluster: "uuid-0",
			},
			fnName: "GetUILogs",
			returns: []interface{}{[]couchbase.UILogEntry{
				{
					Code:       0,
					Module:     "m",
					Node:       "a",
					ServerTime: "a",
					Text:       "hello",
					Type:       "ns",
				},
			}, nil},
		},
		{
			name: "data-loss",
			expectedResult: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "dataLoss",
					Status: values.WarnCheckerStatus,
					Value:  []byte(`{"code":0,"module":"m","node":"a","serverTime":"a","text":"lost data","type":"ns"}`),
				},
				Cluster: "uuid-0",
			},
			checkRemediation: true,
			fnName:           "GetUILogs",
			returns: []interface{}{[]couchbase.UILogEntry{
				{
					Code:       0,
					Module:     "m",
					Node:       "a",
					ServerTime: "a",
					Text:       "lost data",
					Type:       "ns",
				},
			}, nil},
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
					Name:   "backupLocationCheck",
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
					Name:   "backupLocationCheck",
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
					Name:   "backupLocationCheck",
					Status: values.AlertCheckerStatus,
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
		results, err := asymmetricalClusterCheck(cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "asymmetricalCluster",
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
		results, err := asymmetricalClusterCheck(cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "asymmetricalCluster",
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
		results, err := asymmetricalClusterCheck(cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "asymmetricalCluster",
				Time:   cluster.LastUpdate,
				Status: values.WarnCheckerStatus,
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
		results, err := asymmetricalClusterCheck(cluster)
		require.NoError(t, err)
		require.Len(t, results, 1)

		expectedResult := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "asymmetricalCluster",
				Time:   cluster.LastUpdate,
				Status: values.WarnCheckerStatus,
			},
			Cluster: cluster.UUID,
		}

		wrappedResultMustMatch(expectedResult, results[0], true, false, t)
	})
}

// wrappedResultMustMatch gives a bit more information about failures than a reflect.DeepEquals at the top level would.
func wrappedResultMustMatch(expected, value *values.WrappedCheckerResult, shouldHaveRemediation, ignoreTime bool,
	t *testing.T) {
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
