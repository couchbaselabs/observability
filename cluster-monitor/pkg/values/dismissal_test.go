// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import (
	"testing"
	"time"
)

func TestDismissalIsDismissed(t *testing.T) {
	type testCase struct {
		name      string
		dismissal *Dismissal
		result    *WrappedCheckerResult
		expected  bool
	}

	startTime := time.Now()

	cases := []testCase{
		{
			name: "not-the-checker",
			dismissal: &Dismissal{
				Forever:     true,
				Level:       AllDismissLevel,
				CheckerName: "checker-1",
			},
			result: &WrappedCheckerResult{
				Result: &CheckerResult{
					Name: "checker-2",
					Time: time.Now(),
				},
				Cluster: "cluster-0",
			},
		},
		{
			name: "expired",
			dismissal: &Dismissal{
				Until:       startTime.Add(-time.Hour),
				Level:       AllDismissLevel,
				CheckerName: "checker-1",
			},
			result: &WrappedCheckerResult{
				Result: &CheckerResult{
					Name: "checker-1",
					Time: time.Now(),
				},
				Cluster: "cluster-0",
			},
		},
		{
			name: "dismissed-all-level",
			dismissal: &Dismissal{
				Until:       startTime.Add(20 * time.Hour),
				Level:       AllDismissLevel,
				CheckerName: "checker-1",
			},
			result: &WrappedCheckerResult{
				Result: &CheckerResult{
					Name: "checker-1",
					Time: time.Now(),
				},
				Cluster: "cluster-0",
			},
			expected: true,
		},
		{
			name: "dismissed-cluster-level",
			dismissal: &Dismissal{
				Forever:     true,
				Level:       ClusterDismissLevel,
				CheckerName: "checker-1",
				ClusterUUID: "cluster-0",
			},
			result: &WrappedCheckerResult{
				Result: &CheckerResult{
					Name: "checker-1",
					Time: time.Now(),
				},
				Cluster: "cluster-0",
			},
			expected: true,
		},
		{
			name: "dismissed-node-level",
			dismissal: &Dismissal{
				Forever:     true,
				Level:       NodeDismissLevel,
				CheckerName: "checker-1",
				ClusterUUID: "cluster-0",
				NodeUUID:    "node-0",
			},
			result: &WrappedCheckerResult{
				Result: &CheckerResult{
					Name: "checker-1",
					Time: time.Now(),
				},
				Cluster: "cluster-0",
				Node:    "node-0",
			},
			expected: true,
		},
		{
			name: "dismissed-node-level-cluster-dont-match",
			dismissal: &Dismissal{
				Forever:     true,
				Level:       NodeDismissLevel,
				CheckerName: "checker-1",
				ClusterUUID: "cluster-0",
				NodeUUID:    "node-0",
			},
			result: &WrappedCheckerResult{
				Result: &CheckerResult{
					Name: "checker-1",
					Time: time.Now(),
				},
				Cluster: "cluster-1",
				Node:    "node-0",
			},
		},
		{
			name: "dismissed-bucket-level",
			dismissal: &Dismissal{
				Forever:     true,
				Level:       BucketDismissLevel,
				CheckerName: "checker-1",
				ClusterUUID: "cluster-0",
				BucketName:  "bucket-0",
			},
			result: &WrappedCheckerResult{
				Result: &CheckerResult{
					Name: "checker-1",
					Time: time.Now(),
				},
				Cluster: "cluster-0",
				Bucket:  "bucket-0",
			},
			expected: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if out := tc.dismissal.IsDismissed(tc.result); out != tc.expected {
				t.Fatalf("Expected %v got %v", tc.expected, out)
			}
		})
	}
}
