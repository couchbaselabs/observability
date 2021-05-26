package sqlite

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/values"
)

func TestAddAndGetCluster(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	t.Run("no-nodes", func(t *testing.T) {
		cluster := &values.CouchbaseCluster{
			UUID:     "0",
			Name:     "alpha",
			User:     "a",
			Password: "b",
		}

		if err := db.AddCluster(cluster); err == nil {
			t.Fatal("Expected an error but got nil")
		}
	})

	cluster := &values.CouchbaseCluster{
		UUID:     "uuid-0",
		Name:     "c0",
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

	t.Run("valid-add", func(t *testing.T) {
		if err := db.AddCluster(cluster); err != nil {
			t.Fatalf("Unexpected error adding cluster: %v", err)
		}

		// try and get the cluster to confirm it stored the data correctly
		outCluster, err := db.GetCluster("uuid-0", true)
		if err != nil {
			t.Fatalf("Could not get cluster from store: %v", err)
		}

		// this value should be around a minute from the current time
		lastUpdatedTime := outCluster.LastUpdate
		if time.Since(lastUpdatedTime) > time.Minute {
			t.Fatalf("Invalid last update time, expected something closer to the current time got %v", lastUpdatedTime)
		}

		if !compareClusters(cluster, outCluster, true) {
			t.Fatalf("in and otu clusters dont match.\n%+v\n%v", cluster, outCluster)
		}
	})

	t.Run("add-not-unique", func(t *testing.T) {
		if err := db.AddCluster(cluster); err == nil {
			t.Fatalf("Expected an error adding a cluster with the same UUID but got nil")
		}
	})

	t.Run("dont-get-sensitive", func(t *testing.T) {
		outCluster, err := db.GetCluster("uuid-0", false)
		if err != nil {
			t.Fatalf("Expected to be able to get the cluster but got error: %v", err)
		}

		if !compareClusters(cluster, outCluster, false) {
			t.Fatalf("in and otu clusters dont match.\n%+v\n%v", cluster, outCluster)
		}
	})
}

func TestGetClusters(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	// slice has uuids in ascending order
	clusters := []*values.CouchbaseCluster{
		{
			UUID:     "uuid-0",
			Name:     "c0",
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
		},
		{
			UUID:     "uuid-1",
			Name:     "c1",
			User:     "user2",
			Password: "passs",
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
		if err := db.AddCluster(cluster); err != nil {
			t.Fatalf("could not setup cluster data for test: %v", err)
		}
	}

	t.Run("get-sensitive", func(t *testing.T) {
		outClusters, err := db.GetClusters(true)
		if err != nil {
			t.Fatalf("Unexpected error getting clusters: %v", err)
		}

		if len(outClusters) != len(clusters) {
			t.Fatalf("Expected %d clusters fot %d", len(clusters), len(outClusters))
		}

		// the order should be by uuid so it should match the clusters slice
		for i, c := range outClusters {
			if !compareClusters(clusters[i], c, true) {
				t.Fatalf("cluster at position %d don't match.\n%+v\n%+v", i, clusters[i], c)
			}
		}
	})

	t.Run("get", func(t *testing.T) {
		outClusters, err := db.GetClusters(false)
		if err != nil {
			t.Fatalf("Unexpected error getting clusters: %v", err)
		}

		if len(outClusters) != len(clusters) {
			t.Fatalf("Expected %d clusters fot %d", len(clusters), len(outClusters))
		}

		for i, c := range outClusters {
			if !compareClusters(clusters[i], c, false) {
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

	if err := db.AddCluster(cluster); err != nil {
		t.Fatalf("Unexpected error adding cluster: %v", err)
	}

	if err := db.DeleteCluster(cluster.UUID); err != nil {
		t.Fatalf("Unexpected error deleting cluster: %v", err)
	}

	if _, err := db.GetCluster(cluster.UUID, false); !errors.Is(err, values.ErrNotFound) {
		t.Fatalf("Expected a not found error but got %v", err)
	}
}

func TestUpdateCluster(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	original := &values.CouchbaseCluster{
		UUID:     "uuid-0",
		Name:     "c0",
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

	t.Run("update-non-existing", func(t *testing.T) {
		if err := db.UpdateCluster(original); err != nil {
			t.Fatalf("Unexpected error when updating: %v", err)
		}

		clusters, err := db.GetClusters(true)
		if err != nil {
			t.Fatalf("Unexpected error getting clusters: %v", err)
		}

		if len(clusters) != 0 {
			t.Fatalf("Expected zero entries but got %d", len(clusters))
		}
	})

	if err := db.AddCluster(original); err != nil {
		t.Fatalf("Unexpected error adding cluster for test: %v", err)
	}

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
		if err != nil {
			t.Fatalf("Unexpected error updating cluter: %v", err)
		}

		outCluster, err := db.GetCluster(original.UUID, true)
		if err != nil {
			t.Fatalf("unexpected error getting cluster after update: %v", err)
		}

		if !compareClusters(original, outCluster, true) {
			t.Fatalf("in and otu clusters dont match.\n%+v\n%v", original, outCluster)
		}
	})
}

// compareClusters is a helper function to do comparisons between expected clusters and clusters retrieved from the
// store.
func compareClusters(in, out *values.CouchbaseCluster, sensitive bool) bool {
	return in.UUID == out.UUID && in.Name == out.Name && reflect.DeepEqual(in.NodesSummary, out.NodesSummary) &&
		reflect.DeepEqual(in.BucketsSummary, out.BucketsSummary) &&
		reflect.DeepEqual(in.ClusterInfo, out.ClusterInfo) && in.HeartBeatIssue == out.HeartBeatIssue &&
		(!sensitive || (in.User == out.User && in.Password == out.Password))
}
