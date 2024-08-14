// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package janitor

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage/sqlite"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/require"
)

// createAndLoadStore creates a new store in a temporary location and inserts the given dismissals and results.
func createAndLoadStore(t *testing.T, dismissals []values.Dismissal,
	results []*values.WrappedCheckerResult, clusters []*values.CouchbaseCluster,
) storage.Store {
	testDir := t.TempDir()
	store, err := sqlite.NewSQLiteDB(filepath.Join(testDir, "store.db"), "key")
	require.Nil(t, err, "Unexpected error creating test store")

	for _, c := range clusters {
		require.Nil(t, store.AddCluster(c), "could not add results: %v", c)
	}

	for _, r := range results {
		require.Nil(t, store.SetCheckerResult(r), "could not add results: %v", r)
	}

	for _, d := range dismissals {
		require.Nil(t, store.AddDismissal(d), "could not add dismissal: %v", d)
	}

	return store
}

// TestJanitorExpired checks that the janitor cleans up expired dismissals. It does this by adding 3 dismissals to the
// store (An expired one, one that has not expired and one that cannot expire), then runs the janitor for a couple of
// shifts and then stops it. It will check that once the janitor has run that the expired dismissal is no longer in the
// store.
func TestJanitorExpired(t *testing.T) {
	dismissals := []values.Dismissal{
		{
			Until:       time.Now().Add(-24 * time.Hour).UTC(),
			Level:       values.AllDismissLevel,
			ID:          "expired",
			CheckerName: "c0",
		},
		{
			Until:       time.Now().Add(24 * time.Hour).UTC(),
			Level:       values.AllDismissLevel,
			ID:          "alive",
			CheckerName: "c0",
		},
		{
			Forever:     true,
			Level:       values.AllDismissLevel,
			ID:          "forever",
			CheckerName: "c0",
		},
	}

	store := createAndLoadStore(t, dismissals, nil, nil)
	defer store.Close()

	// run janitor for a couple of shifts
	janitor := NewJanitor(store, DefaultConfig)
	janitor.Start(100 * time.Millisecond)
	time.Sleep(500 * time.Millisecond)
	janitor.Stop()

	// check that only the non-expired dismissals are in the store
	outDismissals, err := store.GetDismissals(values.DismissalSearchSpace{})
	require.Nil(t, err)
	require.Len(t, outDismissals, 2)

	for i, d := range outDismissals {
		require.Equal(t, &dismissals[i+1], d, "Values do not match")
	}
}

// TestJanitorUnknownNodes checks that the janitor cleans up dismissals/results for nodes that are no longer part of the
// cluster. It does this by adding 3 dismissals (1 not at node level, 1 for a known node, 1 for an unknown node) and 3
// results with the same pattern as the dismissals. Then it will check that after the janitor shift the data or unknown
// node is no longer there and all the other remains.
func TestJanitorUnknownNodes(t *testing.T) {
	dismissals := []values.Dismissal{
		{
			Until:       time.Now().Add(24 * time.Hour).UTC(),
			Level:       values.NodeDismissLevel,
			ID:          "alive",
			CheckerName: "c0",
			NodeUUID:    "n0",
			ClusterUUID: "uuid-0",
		},
		{
			Until:       time.Now().Add(24 * time.Hour).UTC(),
			Level:       values.AllDismissLevel,
			ID:          "all-level",
			CheckerName: "c0",
		},
		{
			Forever:     true,
			Level:       values.NodeDismissLevel,
			ID:          "forever",
			CheckerName: "c0",
			NodeUUID:    "notFound",
			ClusterUUID: "uuid-0",
		},
	}

	results := []*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:   "c0",
				Status: values.GoodCheckerStatus,
				Time:   time.Now().UTC(),
			},
			Cluster: "uuid-0",
		},
		{
			Result: &values.CheckerResult{
				Name:   "c0",
				Status: values.GoodCheckerStatus,
				Time:   time.Now().UTC(),
			},
			Cluster: "uuid-0",
			Node:    "n0",
		},
		{
			Result: &values.CheckerResult{
				Name:   "c0",
				Status: values.GoodCheckerStatus,
				Time:   time.Now().UTC(),
			},
			Cluster: "uuid-0",
			Node:    "notFound",
		},
	}

	clusters := []*values.CouchbaseCluster{
		{
			UUID:       "uuid-0",
			Enterprise: true,
			User:       "user",
			Password:   "password",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID: "n0",
					Host:     "https://hosts.com",
				},
			},
			LastUpdate: time.Now().UTC(),
		},
	}

	store := createAndLoadStore(t, dismissals, results, clusters)
	defer store.Close()

	// run janitor for a couple of shifts
	janitor := NewJanitor(store, DefaultConfig)
	janitor.Start(100 * time.Millisecond)
	time.Sleep(500 * time.Millisecond)
	janitor.Stop()

	// check that only valid dismissals are in the store
	outDismissals, err := store.GetDismissals(values.DismissalSearchSpace{})
	require.Nil(t, err, "could not get dismissal from store: %v", err)
	require.Len(t, outDismissals, 2, "Expected 2 dismissals")

	for i, d := range outDismissals {
		require.Equal(t, &dismissals[i], d, "Values do not match")
	}

	// check that only valid results are in the store
	outResults, err := store.GetCheckerResult(values.CheckerSearch{Cluster: &clusters[0].UUID})
	require.Nil(t, err, "could not get results from store: %v", err)
	require.Len(t, outResults, 2, "Expected 2 checker results")

	for i, r := range outResults {
		require.Equal(t, results[i], r, "Values do not match")
	}
}

// TestJanitorUnknownBuckets follows the same procedure than TestJanitorUnknownNodes but for buckets.
func TestJanitorUnknownBuckets(t *testing.T) {
	dismissals := []values.Dismissal{
		{
			Until:       time.Now().Add(24 * time.Hour).UTC(),
			Level:       values.NodeDismissLevel,
			ID:          "alive",
			CheckerName: "c0",
			BucketName:  "b0",
			ClusterUUID: "uuid-0",
		},
		{
			Until:       time.Now().Add(24 * time.Hour).UTC(),
			Level:       values.AllDismissLevel,
			ID:          "all-level",
			CheckerName: "c0",
		},
		{
			Forever:     true,
			Level:       values.NodeDismissLevel,
			ID:          "forever",
			CheckerName: "c0",
			BucketName:  "notFound",
			ClusterUUID: "uuid-0",
		},
	}

	results := []*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:   "c0",
				Status: values.GoodCheckerStatus,
				Time:   time.Now().UTC(),
			},
			Cluster: "uuid-0",
		},
		{
			Result: &values.CheckerResult{
				Name:   "c0",
				Status: values.GoodCheckerStatus,
				Time:   time.Now().UTC(),
			},
			Cluster: "uuid-0",
			Bucket:  "b0",
		},
		{
			Result: &values.CheckerResult{
				Name:   "c0",
				Status: values.GoodCheckerStatus,
				Time:   time.Now().UTC(),
			},
			Cluster: "uuid-0",
			Bucket:  "notFound",
		},
	}

	clusters := []*values.CouchbaseCluster{
		{
			UUID:       "uuid-0",
			Enterprise: true,
			User:       "user",
			Password:   "password",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID: "n0",
					Host:     "https://hosts.com",
				},
			},
			BucketsSummary: values.BucketsSummary{
				{
					Name: "b0",
				},
			},
			LastUpdate: time.Now().UTC(),
		},
	}

	store := createAndLoadStore(t, dismissals, results, clusters)
	defer store.Close()

	// run janitor for a couple of shifts
	janitor := NewJanitor(store, DefaultConfig)
	janitor.Start(100 * time.Millisecond)
	time.Sleep(500 * time.Millisecond)
	janitor.Stop()

	// check that only valid dismissals are in the store
	outDismissals, err := store.GetDismissals(values.DismissalSearchSpace{})
	require.Nil(t, err, "could not get dismissal from store: %v", err)
	require.Len(t, outDismissals, 2, "Expected 2 dismissals")

	for i, d := range outDismissals {
		require.Equal(t, &dismissals[i], d, "Values do not match")
	}

	// check that only valid results are in the store
	outResults, err := store.GetCheckerResult(values.CheckerSearch{Cluster: &clusters[0].UUID})
	require.Nil(t, err, "could not get results from store: %v", err)
	require.Len(t, outResults, 2, "Expected 2 checker results")

	for i, r := range outResults {
		require.Equal(t, results[i], r, "Values do not match")
	}
}

func TestJanitorOldLogAlerts(t *testing.T) {
	testCutoff := 5 * time.Minute
	results := []*values.WrappedCheckerResult{
		{
			Result: &values.CheckerResult{
				Name:   values.CheckOOMKills,
				Status: values.WarnCheckerStatus,
				Time:   time.Now().UTC(),
			},
			Cluster: "uuid-0",
		},
		{
			Result: &values.CheckerResult{
				Name:   values.CheckCushionManagedFail,
				Status: values.WarnCheckerStatus,
				Time:   time.Now().Add(-2 * testCutoff).UTC(),
			},
			Cluster: "uuid-0",
		},
	}

	clusters := []*values.CouchbaseCluster{
		{
			UUID:       "uuid-0",
			Enterprise: true,
			User:       "user",
			Password:   "password",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID: "n0",
					Host:     "https://hosts.com",
				},
			},
			BucketsSummary: values.BucketsSummary{
				{
					Name: "b0",
				},
			},
			LastUpdate: time.Now().UTC(),
		},
	}

	store := createAndLoadStore(t, nil, results, clusters)
	defer store.Close()

	// run janitor for a couple of shifts
	janitor := NewJanitor(store, Config{
		LogAlertMaxAge: testCutoff,
	})
	janitor.Start(100 * time.Millisecond)
	time.Sleep(500 * time.Millisecond)
	janitor.Stop()

	// check that only valid results are in the store
	outResults, err := store.GetCheckerResult(values.CheckerSearch{Cluster: &clusters[0].UUID})
	require.Nil(t, err, "could not get results from store: %v", err)
	require.Len(t, outResults, 1, "Expected 1 checker result")

	for i, r := range outResults {
		require.Equal(t, results[i], r, "Values do not match")
	}
}
