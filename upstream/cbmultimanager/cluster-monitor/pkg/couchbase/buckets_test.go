// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/logger"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/aprov"
	"github.com/couchbase/tools-common/cbrest"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func getTestClient(t *testing.T, cluster string) *Client {
	internalClient, err := cbrest.NewClient(cbrest.ClientOptions{
		ConnectionString: cluster,
		Provider:         &aprov.Static{Username: "user", UserAgent: "test-cbmultimanager", Password: "password"},
		DisableCCP:       true,
		Logger:           logger.NewToolsCommonLogger(zap.L().Sugar()),
	})
	require.NoError(t, err)

	return &Client{
		internalClient: internalClient,
		BootstrapTime:  time.Now(),
		ClusterInfo:    &PoolsMetadata{Enterprise: true},
	}
}

func TestGetPoolsBucket(t *testing.T) {
	statusCode := http.StatusOK

	handlers := make(cbrest.TestHandlers)
	handlers.Add(http.MethodGet, string(PoolsBucketEndpoint), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(statusCode, []BucketsEndpointData{
			{
				Name: "b1",
				VBucketServerMap: values.VBucketServerMap{
					ServerList:  []string{"server:1"},
					NumReplicas: 0,
					VBucketMap:  [][]int{{0}, {0}},
				},
			},
			{
				Name: "b2",
				VBucketServerMap: values.VBucketServerMap{
					ServerList:  []string{"server:1", "server:2"},
					NumReplicas: 1,
					VBucketMap:  [][]int{{0, 1}, {1, 0}},
				},
			},
		}, []byte{}, w)
	})

	cluster := cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: true,
		UUID:       "cluster_0",
		Nodes:      cbrest.TestNodes{&cbrest.TestNode{}},
		Handlers:   handlers,
	})
	defer cluster.Close()

	client := getTestClient(t, cluster.URL())

	t.Run("401", func(t *testing.T) {
		statusCode = http.StatusUnauthorized
		_, err := client.GetPoolsBucket()
		if err == nil {
			t.Fatalf("Expected and error but got <nil>")
		}
	})

	t.Run("200", func(t *testing.T) {
		statusCode = http.StatusOK
		expected := []values.Bucket{
			{
				Name: "b1",
				VBucketServerMap: values.VBucketServerMap{
					ServerList:  []string{"server:1"},
					NumReplicas: 0,
					VBucketMap:  [][]int{{0}, {0}},
				},
			},
			{
				Name: "b2",
				VBucketServerMap: values.VBucketServerMap{
					ServerList:  []string{"server:1", "server:2"},
					NumReplicas: 1,
					VBucketMap:  [][]int{{0, 1}, {1, 0}},
				},
			},
		}

		buckets, err := client.GetPoolsBucket()
		if err != nil {
			t.Fatalf("Unexpected error getting buckets: %v", err)
		}

		if !reflect.DeepEqual(expected, buckets) {
			t.Fatalf("Values do not match:\n%+v\n%+v", expected, buckets)
		}
	})
}

func TestGetBucketSummary(t *testing.T) {
	statusCode := http.StatusOK

	handlers := make(cbrest.TestHandlers)
	handlers.Add(http.MethodGet, string(PoolsBucketEndpoint), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(statusCode, []BucketsEndpointData{
			{
				Name:                   "b1",
				CompressionMode:        "active",
				ConflictResolutionType: "seqno",
				BucketType:             "ephemeral",
				StorageBackend:         "none",
				EvictionPolicy:         "nru",
				Quota: struct {
					RAM uint64 `json:"ram"`
				}{RAM: 100},
				BasicStats: struct {
					QuotaPercentUsed float64 `json:"quotaPercentUsed"`
					ItemCount        uint64  `json:"itemCount"`
				}{QuotaPercentUsed: 11.1, ItemCount: 35},
				Controllers: struct {
					Flush string `json:"flush"`
				}{},
				VBucketServerMap: values.VBucketServerMap{
					ServerList:  []string{"server:1"},
					NumReplicas: 0,
					VBucketMap:  [][]int{{0}, {0}},
				},
			},
			{
				Name:                   "b2",
				CompressionMode:        "off",
				ConflictResolutionType: "cas",
				BucketType:             "membase",
				StorageBackend:         "magma",
				EvictionPolicy:         "fullEviction",
				Quota: struct {
					RAM uint64 `json:"ram"`
				}{RAM: 1000},
				BasicStats: struct {
					QuotaPercentUsed float64 `json:"quotaPercentUsed"`
					ItemCount        uint64  `json:"itemCount"`
				}{QuotaPercentUsed: 17.1, ItemCount: 305},
				Controllers: struct {
					Flush string `json:"flush"`
				}{Flush: "flush"},
				VBucketServerMap: values.VBucketServerMap{
					ServerList:  []string{"server:1"},
					NumReplicas: 0,
					VBucketMap:  [][]int{{0}, {0}},
				},
			},
		}, []byte{}, w)
	})

	cluster := cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: true,
		UUID:       "cluster_0",
		Nodes:      cbrest.TestNodes{&cbrest.TestNode{}},
		Handlers:   handlers,
	})
	defer cluster.Close()

	client := getTestClient(t, cluster.URL())

	t.Run("401", func(t *testing.T) {
		statusCode = http.StatusUnauthorized
		_, err := client.GetBucketsSummary()
		require.Error(t, err)
	})

	t.Run("200", func(t *testing.T) {
		statusCode = http.StatusOK
		expected := values.BucketsSummary{
			{
				Name:                   "b1",
				CompressionMode:        "active",
				ConflictResolutionType: "seqno",
				BucketType:             "ephemeral",
				StorageBackend:         "none",
				EvictionPolicy:         "nru",
				Quota:                  100,
				QuotaUsed:              11.1,
				FlushEnabled:           false,
				NumReplicas:            0,
				Items:                  35,
			},
			{
				Name:                   "b2",
				CompressionMode:        "off",
				ConflictResolutionType: "cas",
				BucketType:             "couchbase",
				StorageBackend:         "magma",
				EvictionPolicy:         "fullEviction",
				Quota:                  1000,
				QuotaUsed:              17.1,
				FlushEnabled:           true,
				NumReplicas:            0,
				Items:                  305,
			},
		}

		buckets, err := client.GetBucketsSummary()
		require.NoError(t, err)
		require.Equal(t, expected, buckets)
	})
}

func TestGetBucketStats(t *testing.T) {
	var (
		statusCode int
		data       []byte
	)

	handlers := make(cbrest.TestHandlers)
	handlers.Add(http.MethodGet, string(PoolsBucketStatsEndpoint.Format("bucket")),
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
			_, _ = w.Write(data)
		})

	cluster := cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: true,
		UUID:       "cluster_0",
		Nodes:      cbrest.TestNodes{&cbrest.TestNode{}},
		Handlers:   handlers,
	})
	defer cluster.Close()

	client := getTestClient(t, cluster.URL())

	t.Run("OK", func(t *testing.T) {
		statusCode = http.StatusOK
		data = []byte(`{"op": {"samples": {"vb_active_resident_items_ratio": [7]}}}`)

		res, err := client.GetBucketStats("bucket")
		require.NoError(t, err)
		require.Equal(t, &values.BucketStat{VbActiveRatio: []float64{7}}, res)
	})

	t.Run("NotFound", func(t *testing.T) {
		statusCode = http.StatusNotFound

		_, err := client.GetBucketStats("bucket")
		require.Error(t, err)
		require.ErrorIs(t, err, values.ErrNotFound)
	})
}
