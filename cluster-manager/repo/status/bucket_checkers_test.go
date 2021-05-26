package status

import (
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/couchbase"
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/stretchr/testify/require"
)

func TestMaxBuckets(t *testing.T) {
	type bucketCheckerTest struct {
		name                  string
		buckets               []couchbase.Bucket
		time                  time.Time
		expected              *values.WrappedCheckerResult
		shouldHaveRemediation bool
	}

	cases := []bucketCheckerTest{
		{
			name:    "no-buckets",
			buckets: []couchbase.Bucket{},
			expected: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "maxBuckets",
					Value:  []byte(`{"num_buckets":0}`),
					Status: values.GoodCheckerStatus,
				},
				Cluster: "C0",
			},
		},
		{
			name:    "5-buckets",
			buckets: make([]couchbase.Bucket, 7),
			expected: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "maxBuckets",
					Value:  []byte(`{"num_buckets":7}`),
					Status: values.GoodCheckerStatus,
				},
				Cluster: "C0",
			},
		},
		{
			name:    "31-buckets",
			buckets: make([]couchbase.Bucket, 31),
			expected: &values.WrappedCheckerResult{
				Result: &values.CheckerResult{
					Name:   "maxBuckets",
					Value:  []byte(`{"num_buckets":31}`),
					Status: values.WarnCheckerStatus,
				},
				Cluster: "C0",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wrappedResultMustMatch(tc.expected, maxBuckets(tc.buckets, tc.time, "C0"), tc.shouldHaveRemediation, false,
				t)
		})
	}
}

type bucketsCheckerTest struct {
	name                  string
	buckets               []couchbase.Bucket
	time                  time.Time
	expected              []*values.WrappedCheckerResult
	shouldHaveRemediation bool
}

func runBucketsCheckerTest(t *testing.T, cases []bucketsCheckerTest,
	fn func([]couchbase.Bucket, time.Time, string) []*values.WrappedCheckerResult) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := fn(tc.buckets, tc.time, "C0")

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
			buckets: []couchbase.Bucket{
				{
					Name: "B0",
					VBucketServerMap: couchbase.VBucketServerMap{
						VBucketMap: [][]int{{0, 0}, {0, 0}},
					},
				},
				{
					Name: "B1",
					VBucketServerMap: couchbase.VBucketServerMap{
						VBucketMap: [][]int{{0, 0}, {0, 0}},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "missingActiveVBuckets",
						Value:  []byte("[]"),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "missingActiveVBuckets",
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
			buckets: []couchbase.Bucket{
				{
					Name: "B0",
					VBucketServerMap: couchbase.VBucketServerMap{
						VBucketMap: [][]int{{0, 0}, {-1, 0}, {-1, 0}},
					},
				},
				{
					Name: "B1",
					VBucketServerMap: couchbase.VBucketServerMap{
						// missing replica but it should not care
						VBucketMap: [][]int{{0, 0}, {0, -1}},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "missingActiveVBuckets",
						Value:  []byte("[1,2]"),
						Status: values.AlertCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "missingActiveVBuckets",
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
			buckets: []couchbase.Bucket{
				{
					Name: "B0",
					VBucketServerMap: couchbase.VBucketServerMap{
						NumReplicas: 1,
						// missing active but it should not care
						VBucketMap: [][]int{{0, 0}, {-1, 0}},
					},
				},
				{
					Name: "B1",
					VBucketServerMap: couchbase.VBucketServerMap{
						VBucketMap: [][]int{{0}, {0}},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "missingReplicaVBuckets",
						Value:  []byte("[]"),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "missingReplicaVBuckets",
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
			buckets: []couchbase.Bucket{
				{
					Name: "B0",
					VBucketServerMap: couchbase.VBucketServerMap{
						NumReplicas: 1,
						VBucketMap:  [][]int{{0, -1}, {-1, -1}, {0, 0}},
					},
				},
				{
					Name: "B1",
					VBucketServerMap: couchbase.VBucketServerMap{
						NumReplicas: 1,
						VBucketMap:  [][]int{{0, 0}, {0, 0}},
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "missingReplicaVBuckets",
						Value:  []byte("[0,1]"),
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "missingReplicaVBuckets",
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
	bucketname couchbase.Bucket
	stat       values.BucketStat
}

type bucketCheckerStatsTest struct {
	name                  string
	buckets               []bucketStat
	time                  time.Time
	expected              []*values.WrappedCheckerResult
	shouldHaveRemediation bool
}

func TestBucketMemoryUsage(t *testing.T) {
	cases := []bucketCheckerStatsTest{
		{
			name: "Not enough samples",
			buckets: []bucketStat{
				{
					bucketname: couchbase.Bucket{
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
						Name:   "bucketMemoryUsage",
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
					bucketname: couchbase.Bucket{
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
						Name:   "bucketMemoryUsage",
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
					bucketname: couchbase.Bucket{
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
						Name:   "bucketMemoryUsage",
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
					bucketname: couchbase.Bucket{
						Name: "B0",
					},
					stat: values.BucketStat{
						MemUsed: []float64{6500, 6500, 6500, 6500, 6500},
					},
				},
				{
					bucketname: couchbase.Bucket{
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
						Name:   "bucketMemoryUsage",
						Status: values.WarnCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "bucketMemoryUsage",
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
					bucketname: couchbase.Bucket{
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
						Name:   "residentRatioTooLow",
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
					bucketname: couchbase.Bucket{
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
						Name:   "residentRatioTooLow",
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
					bucketname: couchbase.Bucket{
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
						Name:   "residentRatioTooLow",
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
					bucketname: couchbase.Bucket{
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
						Name:   "residentRatioTooLow",
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
					bucketname: couchbase.Bucket{
						Name: "B0",
					},
					stat: values.BucketStat{
						VbActiveRatio: []float64{70},
					},
				},
				{
					bucketname: couchbase.Bucket{
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
						Name:   "residentRatioTooLow",
						Value:  []byte(`{"residency": 70.00}`),
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "residentRatioTooLow",
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
		buckets               []couchbase.Bucket
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
			},
			buckets: []couchbase.Bucket{
				{
					Name: "B0",
					VBucketServerMap: couchbase.VBucketServerMap{
						NumReplicas: 2,
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "replicavBucketNumber",
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
			},
			buckets: []couchbase.Bucket{
				{
					Name: "B0",
					VBucketServerMap: couchbase.VBucketServerMap{
						NumReplicas: 1,
					},
				},
				{
					Name: "B1",
					VBucketServerMap: couchbase.VBucketServerMap{
						NumReplicas: 3,
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "replicavBucketNumber",
						Status: values.GoodCheckerStatus,
					},
					Cluster: "C0",
					Bucket:  "B0",
				},
				{
					Result: &values.CheckerResult{
						Name:   "replicavBucketNumber",
						Status: values.WarnCheckerStatus,
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
			},
			buckets: []couchbase.Bucket{
				{
					Name: "B0",
					VBucketServerMap: couchbase.VBucketServerMap{
						NumReplicas: 3,
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "replicavBucketNumber",
						Status: values.WarnCheckerStatus,
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
			},
			buckets: []couchbase.Bucket{
				{
					Name: "B0",
					VBucketServerMap: couchbase.VBucketServerMap{
						NumReplicas: 1,
					},
				},
			},
			expected: []*values.WrappedCheckerResult{
				{
					Result: &values.CheckerResult{
						Name:   "replicavBucketNumber",
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
			results := replicavBucketNumber(tc.buckets, tc.time, "C0", len(tc.cluster.NodesSummary))
			require.Len(t, results, len(tc.expected))

			for i, res := range results {
				wrappedResultMustMatch(tc.expected[i], res, tc.shouldHaveRemediation, false, t)
			}
		})
	}
}

func runBucketStatsCheckerTests(t *testing.T, cases []bucketCheckerStatsTest,
	fn func(*values.BucketStat, string, time.Time, string) *values.WrappedCheckerResult) {
	for _, tc := range cases {
		for i := range tc.buckets {
			t.Run(tc.name, func(t *testing.T) {
				result := fn(&tc.buckets[i].stat, tc.buckets[i].bucketname.Name, tc.time, "C0")
				wrappedResultMustMatch(tc.expected[i], result, tc.shouldHaveRemediation, false, t)
			})
		}
	}
}
