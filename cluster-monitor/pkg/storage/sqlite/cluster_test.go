// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"reflect"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/require"
)

func TestAddAndGetCluster(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	t.Run("no-nodes", func(t *testing.T) {
		cluster := &values.CouchbaseCluster{
			UUID:       "0",
			Enterprise: true,
			Name:       "alpha",
			User:       "a",
			Password:   "b",
		}

		require.Error(t, db.AddCluster(cluster))
	})

	cluster := &values.CouchbaseCluster{
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
		BucketsSummary: values.BucketsSummary{
			{
				Name:            "beer-sample",
				Items:           20,
				CompressionMode: "active",
				Quota:           400,
				BucketType:      "membase",
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
	}

	t.Run("valid-add", func(t *testing.T) {
		require.NoError(t, db.AddCluster(cluster))

		// try and get the cluster to confirm it stored the data correctly
		outCluster, err := db.GetCluster("uuid-0", true)
		require.NoError(t, err)

		// this value should be around a minute from the current time
		lastUpdatedTime := outCluster.LastUpdate
		require.LessOrEqualf(t, time.Since(lastUpdatedTime), time.Minute, "last update time expected is to old")

		if !compareClusters(cluster, outCluster, true) {
			t.Fatalf("in and out clusters dont match.\n%+v\n%v", cluster, outCluster)
		}
	})

	t.Run("add-not-unique", func(t *testing.T) {
		require.Error(t, db.AddCluster(cluster))
	})

	t.Run("dont-get-sensitive", func(t *testing.T) {
		outCluster, err := db.GetCluster("uuid-0", false)
		require.NoError(t, err)

		if !compareClusters(cluster, outCluster, false) {
			t.Fatalf("in and out clusters dont match.\n%+v\n%v", cluster, outCluster)
		}
	})

	t.Run("add-ce", func(t *testing.T) {
		cluster.UUID = "ce-uuid"
		cluster.Enterprise = false

		require.NoError(t, db.AddCluster(cluster))
		outCluster, err := db.GetCluster("ce-uuid", true)
		require.NoError(t, err)

		if !compareClusters(cluster, outCluster, true) {
			t.Fatalf("in and out clusters dont match.\n%+v\n%v", cluster, outCluster)
		}
	})

	t.Run("add-with-alias", func(t *testing.T) {
		cluster.UUID = "aliased"
		cluster.Enterprise = true
		cluster.Alias = "a-aka"

		require.NoError(t, db.AddCluster(cluster))
		outCluster, err := db.GetCluster("aliased", true)
		require.NoError(t, err)

		if !compareClusters(cluster, outCluster, true) {
			t.Fatalf("in and out clusters dont match.\n%+v\n%v", cluster, outCluster)
		}
	})
}

func TestGetClusters(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	// slice has uuids in ascending order
	clusters := []*values.CouchbaseCluster{
		{
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
			BucketsSummary: values.BucketsSummary{
				{
					Name:            "beer-sample",
					Items:           20,
					CompressionMode: "active",
					Quota:           400,
					BucketType:      "membase",
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
		},
		{
			UUID:       "uuid-1",
			Name:       "c1",
			Alias:      "a-with-alias",
			Enterprise: true,
			User:       "user2",
			Password:   "passs",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID:          "node4",
					Version:           "7.0.0-0000-enterprise",
					Host:              "alphssa",
					Status:            "healthy",
					ClusterMembership: "active",
					Services:          []string{"kv"},
				},
			},
			BucketsSummary: values.BucketsSummary{
				{
					Name:            "beer-aasample",
					Items:           270,
					CompressionMode: "active",
					Quota:           4010,
					BucketType:      "membase",
				},
			},
			ClusterInfo: &values.ClusterInfo{
				RAMQuota:       90010,
				RAMUsed:        110,
				DiskTotal:      110000,
				DiskUsed:       110,
				DiskUsedByData: 17,
			},
			HeartBeatIssue: values.BadAuthHeartIssue,
		},
		{
			UUID:     "uuid-2",
			Name:     "c3",
			User:     "usear2",
			Password: "passss",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID:          "node4",
					Version:           "7.0.0-0000-enterprise",
					Host:              "alpaahssa",
					Status:            "healthy",
					ClusterMembership: "active",
					Services:          []string{"kv"},
				},
			},
			BucketsSummary: values.BucketsSummary{
				{
					Name:            "beer-aaaasample",
					Items:           270,
					CompressionMode: "active",
					Quota:           4010,
					BucketType:      "membase",
				},
			},
			ClusterInfo: &values.ClusterInfo{
				RAMQuota:       90010,
				RAMUsed:        110,
				DiskTotal:      110000,
				DiskUsed:       110,
				DiskUsedByData: 17,
			},
		},
	}

	for _, cluster := range clusters {
		require.NoError(t, db.AddCluster(cluster))
	}

	t.Run("get-sensitive", func(t *testing.T) {
		outClusters, err := db.GetClusters(true, false)
		require.NoError(t, err)

		require.Equal(t, len(clusters), len(outClusters))

		// the order should be by uuid so it should match the clusters slice
		for i, c := range outClusters {
			if !compareClusters(clusters[i], c, true) {
				t.Fatalf("cluster at position %d don't match.\n%+v\n%+v", i, clusters[i], c)
			}
		}
	})

	t.Run("get", func(t *testing.T) {
		outClusters, err := db.GetClusters(false, false)
		require.NoError(t, err)

		require.Equal(t, len(clusters), len(outClusters))

		for i, c := range outClusters {
			if !compareClusters(clusters[i], c, false) {
				t.Fatalf("cluster at position %d don't match.\n%+v\n%+v", i, clusters[i], c)
			}
		}
	})

	t.Run("get-ee-only", func(t *testing.T) {
		outClusters, err := db.GetClusters(true, true)
		require.NoError(t, err)

		require.Equal(t, len(clusters)-1, len(outClusters))

		for i, c := range outClusters {
			if !compareClusters(clusters[i], c, true) {
				t.Fatalf("cluster at position %d don't match.\n%+v\n%+v", i, clusters[i], c)
			}
		}
	})
}

func TestDeleteCluster(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	cluster := &values.CouchbaseCluster{
		UUID:     "uuid-0",
		Name:     "c0",
		Alias:    "a-33",
		User:     "user",
		Password: "pass",
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
		BucketsSummary: values.BucketsSummary{
			{
				Name:            "beer-sample",
				Items:           20,
				CompressionMode: "active",
				Quota:           400,
				BucketType:      "membase",
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
	}

	require.NoError(t, db.AddCluster(cluster))
	require.NoError(t, db.DeleteCluster(cluster.UUID))

	_, err := db.GetCluster(cluster.UUID, false)
	require.ErrorIs(t, err, values.ErrNotFound)

	_, err = db.GetAlias("a-33")
	require.ErrorIs(t, err, values.ErrNotFound)
}

func TestUpdateCluster(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	original := &values.CouchbaseCluster{
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
		BucketsSummary: values.BucketsSummary{
			{
				Name:            "beer-sample",
				Items:           20,
				CompressionMode: "active",
				Quota:           400,
				BucketType:      "membase",
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
	}

	t.Run("update-non-existing", func(t *testing.T) {
		require.NoError(t, db.UpdateCluster(original))

		clusters, err := db.GetClusters(true, false)
		require.NoError(t, err)
		require.Len(t, clusters, 0)
	})

	require.NoError(t, db.AddCluster(original))

	original.Name = "new-name"
	original.User = "new-user"
	original.Password = "new-password"
	original.NodesSummary = values.NodesSummary{{}}
	original.BucketsSummary = values.BucketsSummary{}
	original.ClusterInfo = &values.ClusterInfo{
		RAMQuota:       100,
		RAMUsed:        20,
		DiskTotal:      100,
		DiskUsed:       10,
		DiskUsedByData: 1,
	}
	original.CaCert = []byte("not a cert but it does not matter")

	t.Run("update-all", func(t *testing.T) {
		err := db.UpdateCluster(original)
		require.NoError(t, err)

		outCluster, err := db.GetCluster(original.UUID, true)
		require.NoError(t, err)

		if !compareClusters(original, outCluster, true) {
			t.Fatalf("in and out clusters dont match.\n%+v\n%v", original, outCluster)
		}
	})
}

// compareClusters is a helper function to do comparisons between expected clusters and clusters retrieved from the
// store.
func compareClusters(in, out *values.CouchbaseCluster, sensitive bool) bool {
	return in.UUID == out.UUID && in.Name == out.Name && reflect.DeepEqual(in.NodesSummary, out.NodesSummary) &&
		in.Enterprise == out.Enterprise && reflect.DeepEqual(in.BucketsSummary, out.BucketsSummary) &&
		reflect.DeepEqual(in.ClusterInfo, out.ClusterInfo) && in.HeartBeatIssue == out.HeartBeatIssue &&
		(!sensitive || (in.User == out.User && in.Password == out.Password))
}
