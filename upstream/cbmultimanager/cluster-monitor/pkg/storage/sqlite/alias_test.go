// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package sqlite

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/require"
)

func TestAddAndGetAlias(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	t.Run("add-alias-cluster-does-not-exist", func(t *testing.T) {
		require.Error(t, db.AddAlias(&values.ClusterAlias{Alias: "a-1", ClusterUUID: "c0"}))
	})

	t.Run("get-alias-does-not-exist", func(t *testing.T) {
		_, err := db.GetAlias("a-1")
		require.ErrorIs(t, err, values.ErrNotFound)
	})

	t.Run("add-alias", func(t *testing.T) {
		require.NoError(t, db.AddCluster(&values.CouchbaseCluster{
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
		}))

		require.NoError(t, db.AddAlias(&values.ClusterAlias{Alias: "a-1", ClusterUUID: "uuid-0"}))

		alias, err := db.GetAlias("a-1")
		require.NoError(t, err)
		require.Equal(t, &values.ClusterAlias{ClusterUUID: "uuid-0", Alias: "a-1"}, alias)
	})

	t.Run("add-repeated-alias-cluster-pair", func(t *testing.T) {
		require.Error(t, db.AddAlias(&values.ClusterAlias{Alias: "a-1", ClusterUUID: "uuid-0"}))
	})

	t.Run("add-repeated-alias", func(t *testing.T) {
		require.NoError(t, db.AddCluster(&values.CouchbaseCluster{
			UUID:       "uuid-1",
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
		}))

		require.Error(t, db.AddAlias(&values.ClusterAlias{Alias: "a-1", ClusterUUID: "uuid-1"}))
	})

	t.Run("add-alias-for-cluster-that-already-has-alias", func(t *testing.T) {
		require.Error(t, db.AddAlias(&values.ClusterAlias{Alias: "a-2", ClusterUUID: "uuid-0"}))
	})
}

func TestDeleteAlias(t *testing.T) {
	db, _ := createEmptyDB(t)
	defer db.Close()

	require.NoError(t, db.AddCluster(&values.CouchbaseCluster{
		UUID:       "uuid-1",
		Alias:      "a-1",
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
	}))

	t.Run("exists", func(t *testing.T) {
		require.NoError(t, db.DeleteAlias("a-1"))
		_, err := db.GetAlias("a-1")
		require.ErrorIs(t, err, values.ErrNotFound)
	})

	t.Run("does-not-exists", func(t *testing.T) {
		require.NoError(t, db.DeleteAlias("a-1"))
	})
}
