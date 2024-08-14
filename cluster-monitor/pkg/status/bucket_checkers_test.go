// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package status

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/memcached"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/memcached/mocks"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/require"
)

func TestMaxBuckets(t *testing.T) {
	type bucketCheckerTest struct {
		name                  string
		cluster               *values.CouchbaseCluster
		time                  time.Time
		expected              *values.WrappedCheckerResult
		shouldHaveRemediation bool
	}

	cases := []bucketCheckerTest{
		{
			name: "no-buckets",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{},
				},
			},
			expected: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckMaxBuckets,
					Value:  []byte(`{"num_buckets":0}`),
					Status: values.GoodCheckerStatus,
				},
				Cluster: "C0",
			},
		},
		{
			name: "5-buckets",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: make([]values.Bucket, 7),
				},
			},
			expected: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckMaxBuckets,
					Value:  []byte(`{"num_buckets":7}`),
					Status: values.GoodCheckerStatus,
				},
				Cluster: "C0",
			},
		},
		{
			name: "31-buckets",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: make([]values.Bucket, 31),
				},
			},
			expected: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   values.CheckMaxBuckets,
					Value:  []byte(`{"num_buckets":31}`),
					Status: values.InfoCheckerStatus,
				},
				Cluster: "C0",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wrappedResultMustMatch(tc.expected, maxBuckets(tc.cluster, tc.time), tc.shouldHaveRemediation, false,
				t)
		})
	}
}

type bucketsCheckerTest struct {
	name                  string
	cluster               *values.CouchbaseCluster
	time                  time.Time
	expected              []*values.WrappedCheckerResult
	shouldHaveRemediation bool
}

func runBucketsCheckerTest(t *testing.T, cases []bucketsCheckerTest,
	fn func(*values.CouchbaseCluster, time.Time) []*values.WrappedCheckerResult,
) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := fn(tc.cluster, tc.time)

			if len(results) != len(tc.expected) {
				t.Fatalf("Expected %d results got %d", len(tc.expected), len(results))
			}

			for i, res := range results {
				wrappedResultMustMatch(tc.expected[i], res, tc.shouldHaveRemediation, false, t)
			}
		})
	}
}

func TestMissingActiveVBuckets(t *testing.T) {
	cases := []bucketsCheckerTest{
		{
			name: "no-missing-vBuckets",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: [][]int{{0, 0}, {0, 0}},
							},
						},
						{
							Name: "B1",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: [][]int{{0, 0}, {0, 0}},
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMissingActiveVBuckets,
						Value:  []byte("[]"),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMissingActiveVBuckets,
						Value:  []byte("[]"),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
			},
		},
		{
			name: "1-missing-vBuckets",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: [][]int{{0, 0}, {-1, 0}, {-1, 0}},
							},
						},
						{
							Name: "B1",
							VBucketServerMap: values.VBucketServerMap{
								// missing replica but it should not care
								VBucketMap: [][]int{{0, 0}, {0, -1}},
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMissingActiveVBuckets,
						Value:  []byte("[1,2]"),
						Status: values.AlertCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMissingActiveVBuckets,
						Value:  []byte("[]"),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
			},
		},
	}

	runBucketsCheckerTest(t, cases, missingActiveVBuckets)
}

func TestMissingVBucketReplicas(t *testing.T) {
	cases := []bucketsCheckerTest{
		{
			name: "no-missing-replicas-vBuckets",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								NumReplicas: 1,
								// missing active but it should not care
								VBucketMap: [][]int{{0, 0}, {-1, 0}},
							},
						},
						{
							Name: "B1",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: [][]int{{0}, {0}},
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMissingReplicaVBuckets,
						Value:  []byte("[]"),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMissingReplicaVBuckets,
						Value:  []byte("[]"),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
			},
		},
		{
			name: "1-missing-vBuckets",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								NumReplicas: 1,
								VBucketMap:  [][]int{{0, -1}, {-1, -1}, {0, 0}},
							},
						},
						{
							Name: "B1",
							VBucketServerMap: values.VBucketServerMap{
								NumReplicas: 1,
								VBucketMap:  [][]int{{0, 0}, {0, 0}},
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMissingReplicaVBuckets,
						Value:  []byte("[0,1]"),
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMissingReplicaVBuckets,
						Value:  []byte("[]"),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
			},
		},
	}

	runBucketsCheckerTest(t, cases, missingVBucketReplicas)
}

type bucketStat struct {
	bucketname values.Bucket
	stat       values.BucketStat
}

type bucketCheckerStatsTest struct {
	name                  string
	nodes                 values.NodesSummary
	buckets               []bucketStat
	time                  time.Time
	expected              []*values.WrappedCheckerResult
	shouldHaveRemediation bool
	DCPStats              []*memcached.DCPMemStats
	VBStat                []*memcached.DefStats
	MemStats              []*memcached.MemoryStats
	VBCheckpointStat      []memcached.BucketCheckpointStats
}

func TestBucketMemoryUsage(t *testing.T) {
	cases := []bucketCheckerStatsTest{
		{
			name: "Not enough samples",
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
					stat: values.BucketStat{
						MemUsed: []float64{2000, 6000},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBucketMemoryUsage,
						Status: values.MissingCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "Good Memory Quota",
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
					stat: values.BucketStat{
						MemUsed: []float64{2000, 6000, 2000, 4000, 2000},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBucketMemoryUsage,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "Bad Memory Quota",
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
					stat: values.BucketStat{
						MemUsed: []float64{6500, 6500, 6500, 6500, 6500},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBucketMemoryUsage,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "Mixed Memory Quota",
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
					stat: values.BucketStat{
						MemUsed: []float64{6500, 6500, 6500, 6500, 6500},
					},
				},
				{
					bucketname: values.Bucket{
						Name: "B1",
					},
					stat: values.BucketStat{
						MemUsed: []float64{1500, 1500, 2500, 1500, 1500},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBucketMemoryUsage,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBucketMemoryUsage,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
			},
		},
	}

	for _, tc := range cases {
		for i := range tc.buckets {
			t.Run(tc.name, func(t *testing.T) {
				result := bucketMemoryUsageCheck(&tc.buckets[i].stat, tc.buckets[i].bucketname.Name,
					tc.time, "C0", 6000)
				wrappedResultMustMatch(tc.expected[i], result, tc.shouldHaveRemediation, false, t)
			})
		}
	}
}

func TestResidentRatioTooLow(t *testing.T) {
	cases := []bucketCheckerStatsTest{
		{
			name: "no residency ratio",
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
					stat: values.BucketStat{
						VbActiveRatio: []float64{},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckResidentRatioTooLow,
						Status: values.MissingCheckerStatus,
						Value:  []byte(`"Missing residency ratio value from REST endpoint. Will re-run soon."`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "resident ratio warn",
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
					stat: values.BucketStat{
						VbActiveRatio: []float64{7},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckResidentRatioTooLow,
						Value:  []byte(`{"residency": 7.00}`),
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "resident ratio alert",
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
					stat: values.BucketStat{
						VbActiveRatio: []float64{1},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckResidentRatioTooLow,
						Value:  []byte(`{"residency": 1.00}`),
						Status: values.AlertCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "resident ratio good",
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
					stat: values.BucketStat{
						VbActiveRatio: []float64{70},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckResidentRatioTooLow,
						Value:  []byte(`{"residency": 70.00}`),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "resident ratio mixed",
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
					stat: values.BucketStat{
						VbActiveRatio: []float64{70},
					},
				},
				{
					bucketname: values.Bucket{
						Name: "B1",
					},
					stat: values.BucketStat{
						VbActiveRatio: []float64{3},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckResidentRatioTooLow,
						Value:  []byte(`{"residency": 70.00}`),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckResidentRatioTooLow,
						Value:  []byte(`{"residency": 3.00}`),
						Status: values.AlertCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
			},
		},
	}

	runBucketStatsCheckerTests(t, cases, residentRatioTooLowCheck)
}

func TestReplicavBucketNumber(t *testing.T) {
	type bucketCheckerTest struct {
		name                  string
		time                  time.Time
		expected              []*values.WrappedCheckerResult
		cluster               *values.CouchbaseCluster
		shouldHaveRemediation bool
	}
	cases := []bucketCheckerTest{
		{
			name: "10-nodes-2-replica",
			cluster: &values.CouchbaseCluster{
				NodesSummary: values.NodesSummary{
					values.NodeSummary{NodeUUID: "1"},
					values.NodeSummary{NodeUUID: "6"},
					values.NodeSummary{NodeUUID: "2"},
					values.NodeSummary{NodeUUID: "3"},
					values.NodeSummary{NodeUUID: "4"},
					values.NodeSummary{NodeUUID: "5"},
					values.NodeSummary{NodeUUID: "6"},
					values.NodeSummary{NodeUUID: "7"},
					values.NodeSummary{NodeUUID: "8"},
					values.NodeSummary{NodeUUID: "9"},
					values.NodeSummary{NodeUUID: "10"},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								NumReplicas: 2,
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckReplicaVBucketNumber,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "5-nodes-mixed",
			cluster: &values.CouchbaseCluster{
				NodesSummary: values.NodesSummary{
					values.NodeSummary{NodeUUID: "1"},
					values.NodeSummary{NodeUUID: "2"},
					values.NodeSummary{NodeUUID: "3"},
					values.NodeSummary{NodeUUID: "4"},
					values.NodeSummary{NodeUUID: "5"},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								NumReplicas: 1,
							},
						},
						{
							Name: "B1",
							VBucketServerMap: values.VBucketServerMap{
								NumReplicas: 3,
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckReplicaVBucketNumber,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   values.CheckReplicaVBucketNumber,
						Status: values.InfoCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
			},
		},
		{
			name: "1-nodes-3-replica",
			cluster: &values.CouchbaseCluster{
				NodesSummary: values.NodesSummary{
					values.NodeSummary{NodeUUID: "1"},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								NumReplicas: 3,
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckReplicaVBucketNumber,
						Status: values.InfoCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "1-nodes-1-replica",
			cluster: &values.CouchbaseCluster{
				NodesSummary: values.NodesSummary{
					values.NodeSummary{NodeUUID: "1"},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								NumReplicas: 1,
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckReplicaVBucketNumber,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buckets, _ := tc.cluster.GetCacheBuckets()
			results := replicavBucketNumber(buckets, tc.time, "C0", len(tc.cluster.NodesSummary))
			require.Len(t, results, len(tc.expected))

			for i, res := range results {
				wrappedResultMustMatch(tc.expected[i], res, tc.shouldHaveRemediation, false, t)
			}
		})
	}
}

func TestMemcachedFragCheck(t *testing.T) {
	cases := []bucketCheckerStatsTest{
		{
			name: "goodSeven",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "7.0.2-6703-enterprise",
					Host:     "http://H1:8091",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMemcachedFragmentation,
						Status: values.GoodCheckerStatus,
						Value:  []byte(`{"http://H1:8091":"10.00%"}`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			MemStats: []*memcached.MemoryStats{
				{
					ArenaFragmentationBytes: "10",
					ArenaResidentBytes:      "100",
					Host:                    "http://H1:8091",
				},
			},
		},
		{
			name: "warnSeven",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "7.0.2-6703-enterprise",
					Host:     "http://H1:8091",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMemcachedFragmentation,
						Status: values.WarnCheckerStatus,
						Value:  []byte(`{"http://H1:8091":"23.00%"}`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			MemStats: []*memcached.MemoryStats{
				{
					ArenaFragmentationBytes: "23",
					ArenaResidentBytes:      "100",
					Host:                    "http://H1:8091",
				},
			},
		},
		{
			name: "alertSeven",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "7.0.2-6703-enterprise",
					Host:     "http://H1:8091",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMemcachedFragmentation,
						Status: values.AlertCheckerStatus,
						Value:  []byte(`{"http://H1:8091":"30.00%"}`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			MemStats: []*memcached.MemoryStats{
				{
					ArenaFragmentationBytes: "30",
					ArenaResidentBytes:      "100",
					Host:                    "http://H1:8091",
				},
			},
		},
		{
			name: "goodSix",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "6.6.3-9808-enterprise",
					Host:     "http://H1:8091",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMemcachedFragmentation,
						Status: values.GoodCheckerStatus,
						Value:  []byte(`{"http://H1:8091":"10.00%"}`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			MemStats: []*memcached.MemoryStats{
				{
					FragmentationBytes: "10",
					HeapBytes:          "100",
					Host:               "http://H1:8091",
				},
			},
		},
		{
			name: "warnSix",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "6.6.3-9808-enterprise",
					Host:     "http://H1:8091",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMemcachedFragmentation,
						Status: values.WarnCheckerStatus,
						Value:  []byte(`{"http://H1:8091":"17.00%"}`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			MemStats: []*memcached.MemoryStats{
				{
					FragmentationBytes: "17",
					HeapBytes:          "100",
					Host:               "http://H1:8091",
				},
			},
		},
		{
			name: "alertSix",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "6.6.3-9808-enterprise",
					Host:     "http://H1:8091",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMemcachedFragmentation,
						Status: values.AlertCheckerStatus,
						Value:  []byte(`{"http://H1:8091":"30.00%"}`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			MemStats: []*memcached.MemoryStats{
				{
					FragmentationBytes: "30",
					HeapBytes:          "100",
					Host:               "http://H1:8091",
				},
			},
		},
		{
			name: "goodMixedVersion",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "7.0.2-6703-enterprise",
					Host:     "http://H1:8091",
				},
				{
					NodeUUID: "N1",
					Version:  "6.6.3-9808-enterprise",
					Host:     "http://H2:8091",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMemcachedFragmentation,
						Status: values.GoodCheckerStatus,
						Value:  []byte(`{"http://H1:8091":"10.00%","http://H2:8091":"9.00%"}`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			MemStats: []*memcached.MemoryStats{
				{
					ArenaFragmentationBytes: "10",
					ArenaResidentBytes:      "100",
					Host:                    "http://H1:8091",
				},
				{
					FragmentationBytes: "9",
					HeapBytes:          "100",
					Host:               "http://H2:8091",
				},
			},
		},
		{
			name: "warnMixedVersion",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "7.0.2-6703-enterprise",
					Host:     "http://H1:8091",
				},
				{
					NodeUUID: "N1",
					Version:  "6.6.3-9808-enterprise",
					Host:     "http://H2:8091",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMemcachedFragmentation,
						Status: values.WarnCheckerStatus,
						Value:  []byte(`{"http://H1:8091":"22.00%","http://H2:8091":"16.00%"}`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			MemStats: []*memcached.MemoryStats{
				{
					ArenaFragmentationBytes: "22",
					ArenaResidentBytes:      "100",
					Host:                    "http://H1:8091",
				},
				{
					FragmentationBytes: "16",
					HeapBytes:          "100",
					Host:               "http://H2:8091",
				},
			},
		},
		{
			name: "alertMixedVersion",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "7.0.2-6703-enterprise",
					Host:     "http://H1:8091",
				},
				{
					NodeUUID: "N1",
					Version:  "6.6.3-9808-enterprise",
					Host:     "http://H2:8091",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckMemcachedFragmentation,
						Status: values.AlertCheckerStatus,
						Value:  []byte(`{"http://H1:8091":"32.00%","http://H2:8091":"21.00%"}`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			MemStats: []*memcached.MemoryStats{
				{
					ArenaFragmentationBytes: "32",
					ArenaResidentBytes:      "100",
					Host:                    "http://H1:8091",
				},
				{
					FragmentationBytes: "21",
					HeapBytes:          "100",
					Host:               "http://H2:8091",
				},
			},
		},
	}
	for _, tc := range cases {
		for i, bucket := range tc.buckets {
			t.Run(tc.name, func(t *testing.T) {
				client := new(mocks.ConnIFace)

				cluster := &values.CouchbaseCluster{
					UUID:         "C0",
					Name:         "cluster",
					LastUpdate:   time.Now().UTC(),
					NodesSummary: tc.nodes,
				}

				client.On("MemStats", bucket.bucketname.Name).Return(tc.MemStats, nil)
				result, _ := memcachedFragCheck(bucket.bucketname.Name, cluster, client)
				wrappedResultMustMatch(tc.expected[i], result, tc.shouldHaveRemediation, true, t)
			})
		}
	}
}

func TestBucketDCPQueue(t *testing.T) {
	cases := []bucketCheckerStatsTest{
		{
			name: "good",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "6.6.0-0000-enterprise",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBucketDCPPaused,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			DCPStats: []*memcached.DCPMemStats{
				{
					Host: "H1",
				},
			},
			VBStat: []*memcached.DefStats{
				{
					VbActiveSyncAccepted: "0",
					Host:                 "H1",
				},
			},
		},
		{
			name: "good-nonZeroVB",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "6.6.0-0000-enterprise",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBucketDCPPaused,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			DCPStats: []*memcached.DCPMemStats{
				{
					Host: "H1",
					MaxBufferBytes: []memcached.ReplicationStat{
						{
							Value: "1000",
						},
					},
					PausedReason: []memcached.ReplicationStat{
						{
							Extras: "PausedReason::Ready",
						},
					},
				},
			},
			VBStat: []*memcached.DefStats{
				{
					VbActiveSyncAccepted: "0",
					Host:                 "H1",
				},
			},
		},
		{
			name: "bad-PausedBuffer",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "6.6.0-0000-enterprise",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBucketDCPPaused,
						Status: values.AlertCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			DCPStats: []*memcached.DCPMemStats{
				{
					Host: "H1",
					MaxBufferBytes: []memcached.ReplicationStat{
						{
							Value: "1000",
						},
					},
					PausedReason: []memcached.ReplicationStat{
						{
							Extras: "PausedReason::BufferLogFull",
						},
					},
				},
			},
			VBStat: []*memcached.DefStats{
				{
					VbActiveSyncAccepted: "1000",
					Host:                 "H1",
				},
			},
		},
		{
			name: "warn",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "6.6.0-0000-enterprise",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBucketDCPPaused,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			DCPStats: []*memcached.DCPMemStats{
				{
					Host: "H1",
					MaxBufferBytes: []memcached.ReplicationStat{
						{
							Value: "1000",
						},
					},
					PausedReason: []memcached.ReplicationStat{
						{
							Extras: "PausedReason::Ready",
						},
					},
				},
			},
			VBStat: []*memcached.DefStats{
				{
					VbActiveSyncAccepted: "1000",
					Host:                 "H1",
				},
			},
		},
		{
			name: "good-fixedVersion",
			nodes: values.NodesSummary{
				{
					NodeUUID: "N0",
					Version:  "6.6.3-0000-enterprise",
				},
			},
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckBucketDCPPaused,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			DCPStats: []*memcached.DCPMemStats{
				{
					Host: "H1",
					MaxBufferBytes: []memcached.ReplicationStat{
						{
							Value: "1000",
						},
					},
					PausedReason: []memcached.ReplicationStat{
						{
							Extras: "PausedReason::Ready",
						},
					},
				},
			},
			VBStat: []*memcached.DefStats{
				{
					VbActiveSyncAccepted: "1000",
					Host:                 "H1",
				},
			},
		},
	}

	for _, tc := range cases {
		for i, bucket := range tc.buckets {
			t.Run(tc.name, func(t *testing.T) {
				client := new(mocks.ConnIFace)

				cluster := &values.CouchbaseCluster{
					UUID:         "C0",
					Name:         "cluster",
					LastUpdate:   time.Now().UTC(),
					NodesSummary: tc.nodes,
				}

				client.On("DCPStats", bucket.bucketname.Name).Return(tc.DCPStats, nil)
				client.On("DefaultStats", bucket.bucketname.Name).Return(tc.VBStat, nil)

				result := checkMB46482(tc.DCPStats, cluster, bucket.bucketname.Name, client, tc.time)
				wrappedResultMustMatch(tc.expected[i], result, tc.shouldHaveRemediation, false, t)
			})
		}
	}
}

func TestLargeCheckpoints(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
		NodesSummary: values.NodesSummary{
			{
				NodeUUID: "N0",
			},
		},
		BucketsSummary: values.BucketsSummary{
			{
				Name:  "B0",
				Quota: uint64(512 * 1024 * 1024),
			},
		},
	}

	testTime, _ := time.Parse(time.RFC3339, "2021-09-23T10:25:00+01:00")
	cases := []bucketCheckerStatsTest{
		{
			name: "good",
			time: testTime,
			buckets: []bucketStat{
				{bucketname: values.Bucket{Name: "B0"}},
			},
			VBCheckpointStat: []memcached.BucketCheckpointStats{
				{map[string]string{"mem_usage": "0"}},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckLargeCheckpoints,
						Status: values.GoodCheckerStatus,
						Time:   testTime,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "warn",
			time: testTime,
			buckets: []bucketStat{
				{bucketname: values.Bucket{Name: "B0"}},
			},
			VBCheckpointStat: []memcached.BucketCheckpointStats{
				{map[string]string{"mem_usage": strconv.Itoa(100 * 1_000_000)}},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckLargeCheckpoints,
						Status: values.WarnCheckerStatus,
						Time:   testTime,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			shouldHaveRemediation: true,
		},
	}

	for _, tc := range cases {
		for i, bucket := range tc.buckets {
			t.Run(tc.name, func(t *testing.T) {
				client := new(mocks.ConnIFace)
				client.On("Hosts").Return([]string{"H0"})
				client.On("CheckpointStats", "H0", bucket.bucketname.Name).Return(tc.VBCheckpointStat[i], nil)

				result := largeCheckpointsCheck(bucket.bucketname.Name, testTime, cluster, client)
				wrappedResultMustMatch(tc.expected[i], result, tc.shouldHaveRemediation, false, t)
			})
		}
	}
}

var twoHundredChars = strings.Repeat("a", 200)

func TestLongDCPNames(t *testing.T) {
	cluster := &values.CouchbaseCluster{
		UUID:       "C0",
		Name:       "cluster",
		LastUpdate: time.Now().UTC(),
		NodesSummary: values.NodesSummary{
			{
				NodeUUID: "N0",
			},
		},
		BucketsSummary: values.BucketsSummary{
			{
				Name:  "B0",
				Quota: uint64(512 * 1024 * 1024),
			},
		},
	}

	testTime, _ := time.Parse(time.RFC3339, "2021-11-12T11:47:00Z")
	cases := []bucketCheckerStatsTest{
		{
			name: "good",
			time: testTime,
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			DCPStats: []*memcached.DCPMemStats{
				{
					StreamNames: []string{
						`replication:ns_1@10.240.0.6->ns_1@10.240.0.8:travel-sample`,
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckLongDCPStreamNames,
						Status: values.GoodCheckerStatus,
						Value:  []byte(`null`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "alertSynthetic",
			time: testTime,
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			DCPStats: []*memcached.DCPMemStats{
				{
					StreamNames: []string{
						`replication:` + twoHundredChars,
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckLongDCPStreamNames,
						Status: values.WarnCheckerStatus,
						Value:  []byte(`["replication:` + twoHundredChars + `"]`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			shouldHaveRemediation: true,
		},
		{
			name: "alertRealistic",
			time: testTime,
			buckets: []bucketStat{
				{
					bucketname: values.Bucket{
						Name: "B0",
					},
				},
			},
			DCPStats: []*memcached.DCPMemStats{
				{
					StreamNames: []string{
						"replication:ns_1@couchbase-cluster-couchbase-cluster-0001." +
							"couchbase-cluster-couchbase-cluster.couchbase-operator." +
							"svc->ns_1@couchbase-cluster-couchbase-cluster-0002.couchbase-cluster-couchbase-cluster." +
							"couchbase-operator.svc:travel-sample",
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckLongDCPStreamNames,
						Status: values.WarnCheckerStatus,
						Value: []byte(`["` + "replication:ns_1@couchbase-cluster-couchbase-cluster-0001." +
							"couchbase-cluster-couchbase-cluster.couchbase-operator." +
							"svc-\\u003ens_1@couchbase-cluster-couchbase-cluster-0002.couchbase-cluster-couchbase-cluster." +
							"couchbase-operator.svc:travel-sample" + `"]`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			shouldHaveRemediation: true,
		},
	}

	for _, tc := range cases {
		for i, bucket := range tc.buckets {
			t.Run(fmt.Sprintf("%s@%s", tc.name, bucket.bucketname.Name), func(t *testing.T) {
				result := checkMB34280(tc.DCPStats, cluster, bucket.bucketname.Name, tc.time)
				wrappedResultMustMatch(tc.expected[i], result, tc.shouldHaveRemediation, false, t)
			})
		}
	}
}

func runBucketStatsCheckerTests(t *testing.T, cases []bucketCheckerStatsTest,
	fn func(*values.BucketStat, string, time.Time, string) *values.WrappedCheckerResult,
) {
	for _, tc := range cases {
		for i := range tc.buckets {
			t.Run(tc.name, func(t *testing.T) {
				result := fn(&tc.buckets[i].stat, tc.buckets[i].bucketname.Name, tc.time, "C0")
				wrappedResultMustMatch(tc.expected[i], result, tc.shouldHaveRemediation, false, t)
			})
		}
	}
}

func TestUnknownStorageEngine(t *testing.T) {
	cases := []bucketsCheckerTest{
		{
			name: "unknown storage engine",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name:          "B0",
							StorageEngine: "non-standard",
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckUnknownStorageEngine,
						Status: values.AlertCheckerStatus,
						Value:  []byte(`"non-standard"`),
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			shouldHaveRemediation: true,
		},
		{
			name: "known storage engine (couchstore)",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name:          "B0",
							StorageEngine: "couchstore",
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckUnknownStorageEngine,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "known storage engine (ephemeral)",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name:          "B0",
							StorageEngine: "ephemeral",
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckUnknownStorageEngine,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "known storage engine (magma)",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name:          "B0",
							StorageEngine: "magma",
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   values.CheckUnknownStorageEngine,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
		},
		{
			name: "storage engine stat doesn't exist (<7.0)",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name:          "B0",
							StorageEngine: "",
						},
					},
				},
			},
			expected: nil,
		},
	}
	runBucketsCheckerTest(t, cases, unknownStorageEngineCheck)
}

func TestHistogramUnderflowCheck(t *testing.T) {
	cases := []struct {
		name                  string
		cluster               *values.CouchbaseCluster
		stats                 []*memcached.DefStats
		expected              map[string]*values.WrappedCheckerResult
		shouldHaveRemediation bool
	}{
		{
			name: "safe",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				NodesSummary: []values.NodeSummary{
					{
						NodeUUID: "N0",
						Version:  "6.0.0",
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
						},
					},
				},
			},
			stats: []*memcached.DefStats{
				{
					CmdGet: "0",
					CmdSet: "0",
				},
			},
			expected: map[string]*values.WrappedCheckerResult{
				"B0": {
					Result: &values.CheckerResult{
						Name:   values.CheckHistogramUnderflow,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			shouldHaveRemediation: false,
		},
		{
			name: "vulnerable",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				NodesSummary: []values.NodeSummary{
					{
						NodeUUID: "N0",
						Version:  "6.5.1",
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
						},
					},
				},
			},
			stats: []*memcached.DefStats{
				{
					CmdGet: "0",
					CmdSet: "0",
				},
			},
			expected: map[string]*values.WrappedCheckerResult{
				"B0": {
					Result: &values.CheckerResult{
						Name:   values.CheckHistogramUnderflow,
						Status: values.InfoCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			shouldHaveRemediation: false,
		},
		{
			name: "approaching",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				NodesSummary: []values.NodeSummary{
					{
						NodeUUID: "N0",
						Version:  "6.5.1",
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
						},
					},
				},
			},
			stats: []*memcached.DefStats{
				{
					CmdGet: "2000000000",
					CmdSet: "0",
				},
			},
			expected: map[string]*values.WrappedCheckerResult{
				"B0": {
					Result: &values.CheckerResult{
						Name:   values.CheckHistogramUnderflow,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			shouldHaveRemediation: false,
		},
		{
			name: "bad",
			cluster: &values.CouchbaseCluster{
				UUID: "C0",
				NodesSummary: []values.NodeSummary{
					{
						NodeUUID: "N0",
						Version:  "6.5.1",
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
						},
					},
				},
			},
			stats: []*memcached.DefStats{
				{
					CmdGet: "3000000000",
					CmdSet: "0",
				},
			},
			expected: map[string]*values.WrappedCheckerResult{
				"B0": {
					Result: &values.CheckerResult{
						Name:   values.CheckHistogramUnderflow,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			shouldHaveRemediation: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := new(mocks.ConnIFace)
			buckets, _ := tc.cluster.GetCacheBuckets()

			bucket := buckets[0]
			client.On("DefaultStats", bucket.Name, mock.Anything).Return(tc.stats, nil)

			callTime := time.Now()
			result := histogramUnderflowCheck(bucket.Name, callTime, tc.cluster, client)
			wrappedResultMustMatch(tc.expected[bucket.Name], result, tc.shouldHaveRemediation, true, t)
		})
	}
}

func TestMaxTTLBucket(t *testing.T) {
	type bucketWithTTLCheckerTest struct {
		name                  string
		cluster               values.CouchbaseCluster
		time                  time.Time
		expected              []*values.WrappedCheckerResult
		shouldHaveRemediation []bool
	}

	testTime, _ := time.Parse(time.RFC3339, "2021-11-12T11:47:00Z")
	cases := []bucketWithTTLCheckerTest{
		{
			name: "checkLessThan30DaysTTL",
			time: testTime,
			cluster: values.CouchbaseCluster{
				UUID:       "C0",
				Name:       "cluster",
				LastUpdate: time.Now().UTC(),
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						Version:  "5.5.1",
						Uptime:   "1000",
					},
					{
						NodeUUID: "N1",
						Version:  "5.5.1",
						Uptime:   "2000",
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name:   "B0",
							MaxTTL: 10000,
						},
						{
							Name:   "B1",
							MaxTTL: 0,
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckMaxTTLBucket,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckMaxTTLBucket,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
			},
			shouldHaveRemediation: []bool{
				false,
				false,
			},
		},
		{
			name: "checkGreaterThanEqualTo30DaysTTL",
			time: testTime,
			cluster: values.CouchbaseCluster{
				UUID:       "C0",
				Name:       "cluster",
				LastUpdate: time.Now().UTC(),
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						Version:  "5.5.1",
						Uptime:   "1000",
					},
					{
						NodeUUID: "N1",
						Version:  "5.5.1",
						Uptime:   "27000",
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name:   "B0",
							MaxTTL: 30 * 24 * 60 * 60,
						},
						{
							Name:   "B1",
							MaxTTL: 30*24*60*60 + 1,
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckMaxTTLBucket,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckMaxTTLBucket,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
			},
			shouldHaveRemediation: []bool{
				true,
				true,
			},
		},
		{
			name: "checkGreaterThanEqualTo30DaysTTLAndUptime",
			time: testTime,
			cluster: values.CouchbaseCluster{
				UUID:       "C0",
				Name:       "cluster",
				LastUpdate: time.Now().UTC(),
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						Version:  "5.5.1",
						Uptime:   "1000",
					},
					{
						NodeUUID: "N1",
						Version:  "5.5.1",
						Uptime:   "2700001",
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name:   "B0",
							MaxTTL: 2592000,
						},
						{
							Name:   "B1",
							MaxTTL: 2700000,
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckMaxTTLBucket,
						Status: values.AlertCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckMaxTTLBucket,
						Status: values.AlertCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
			},
			shouldHaveRemediation: []bool{
				true,
				true,
			},
		},
		{
			name: "checkVersion",
			time: testTime,
			cluster: values.CouchbaseCluster{
				UUID:       "C0",
				Name:       "cluster",
				LastUpdate: time.Now().UTC(),
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						Version:  "6.0.4",
						Uptime:   "1000",
					},
					{
						NodeUUID: "N1",
						Version:  "6.0.4",
						Uptime:   "2700000",
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name:   "B0",
							MaxTTL: 2592000,
						},
						{
							Name:   "B1",
							MaxTTL: 2700000,
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckMaxTTLBucket,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckMaxTTLBucket,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
			},
			shouldHaveRemediation: []bool{
				false,
				false,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checkMB37643(&tc.cluster, tc.time)
			for i := range result {
				wrappedResultMustMatch(tc.expected[i], result[i], tc.shouldHaveRemediation[i], false, t)
			}
		})
	}
}

func createVBucketMap(length int) [][]int {
	vBucketMap := make([][]int, 0, length)

	for i := 0; i < length; i++ {
		vBucketMap = append(vBucketMap, []int{0, 1})
	}

	return vBucketMap
}

func TestNonDefaultVBucketNumber(t *testing.T) {
	type nonDeafultVBucketNumber struct {
		name                  string
		cluster               values.CouchbaseCluster
		time                  time.Time
		expected              []*values.WrappedCheckerResult
		shouldHaveRemediation []bool
	}

	testTime, _ := time.Parse(time.RFC3339, "2021-11-12T11:47:00Z")
	cases := []nonDeafultVBucketNumber{
		{
			name: "checkForMac",
			time: testTime,
			cluster: values.CouchbaseCluster{
				UUID:       "C0",
				Name:       "cluster",
				LastUpdate: time.Now().UTC(),
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						OS:       "x86_64-apple-darwin18.7.0",
					},
					{
						NodeUUID: "N1",
						OS:       "x86_64-apple-darwin18.7.0",
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: createVBucketMap(1024),
							},
						},
						{
							Name: "B1",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: createVBucketMap(512),
							},
						},
						{
							Name: "B2",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: createVBucketMap(64),
							},
						},
						{
							Name: "B3",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: createVBucketMap(31),
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckDefaultVBucketCount,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckDefaultVBucketCount,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckDefaultVBucketCount,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B2",
				},
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckDefaultVBucketCount,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B3",
				},
			},
			shouldHaveRemediation: []bool{
				true,
				true,
				false,
				true,
			},
		},
		{
			name: "checkForWindowsAndLinux",
			time: testTime,
			cluster: values.CouchbaseCluster{
				UUID:       "C0",
				Name:       "cluster",
				LastUpdate: time.Now().UTC(),
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						OS:       "linux",
					},
					{
						NodeUUID: "N1",
						OS:       "linux",
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: createVBucketMap(1024),
							},
						},
						{
							Name: "B1",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: createVBucketMap(512),
							},
						},
						{
							Name: "B2",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: createVBucketMap(64),
							},
						},
						{
							Name: "B3",
							VBucketServerMap: values.VBucketServerMap{
								VBucketMap: createVBucketMap(31),
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckDefaultVBucketCount,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckDefaultVBucketCount,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B1",
				},
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckDefaultVBucketCount,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B2",
				},
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckDefaultVBucketCount,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B3",
				},
			},
			shouldHaveRemediation: []bool{
				false,
				true,
				true,
				true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checkDefaultVBucketCount(&tc.cluster, tc.time)
			for i := range result {
				wrappedResultMustMatch(tc.expected[i], result[i], tc.shouldHaveRemediation[i], false, t)
			}
		})
	}
}

func TestNodesForBucket(t *testing.T) {
	type nonDeafultVBucketNumber struct {
		name                  string
		cluster               values.CouchbaseCluster
		time                  time.Time
		expected              []*values.WrappedCheckerResult
		shouldHaveRemediation bool
	}

	testTime, _ := time.Parse(time.RFC3339, "2021-11-12T11:47:00Z")
	cases := []nonDeafultVBucketNumber{
		{
			name: "good",
			time: testTime,
			cluster: values.CouchbaseCluster{
				UUID:       "C0",
				Name:       "cluster",
				LastUpdate: time.Now().UTC(),
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						Services: []string{"kv", "index"},
					},
					{
						NodeUUID: "N1",
						Services: []string{"kv", "index"},
					},
					{
						NodeUUID: "N1",
						Services: []string{"index"},
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								ServerList: []string{"1.1.1.1", "2.2.2.2"},
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckNodesForBucket,
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			shouldHaveRemediation: false,
		},
		{
			name: "unequal servers - bad",
			time: testTime,
			cluster: values.CouchbaseCluster{
				UUID:       "C0",
				Name:       "cluster",
				LastUpdate: time.Now().UTC(),
				NodesSummary: values.NodesSummary{
					{
						NodeUUID: "N0",
						Services: []string{"kv", "index"},
					},
					{
						NodeUUID: "N1",
						Services: []string{"kv", "index"},
					},
					{
						NodeUUID: "N1",
						Services: []string{"kv", "index"},
					},
				},
				CacheRESTData: values.CacheRESTData{
					Buckets: []values.Bucket{
						{
							Name: "B0",
							VBucketServerMap: values.VBucketServerMap{
								ServerList: []string{"1.1.1.1", "2.2.2.2"},
							},
						},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Time:   testTime,
						Name:   values.CheckNodesForBucket,
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
			},
			shouldHaveRemediation: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checkNodesForBucket(&tc.cluster, tc.time)
			for i := range result {
				wrappedResultMustMatch(tc.expected[i], result[i], tc.shouldHaveRemediation, false, t)
			}
		})
	}
}
