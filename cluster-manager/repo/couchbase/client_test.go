package couchbase

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/slice"
	"github.com/stretchr/testify/require"
)

func TestNewClientError(t *testing.T) {
	handlers := make(cbrest.TestHandlers)

	var errorCode, count int
	var body []byte
	handlers.Add(http.MethodGet, string(cbrest.EndpointNodesServices), func(w http.ResponseWriter, _ *http.Request) {
		count++
		w.WriteHeader(errorCode)
		_, _ = w.Write(body)
	})

	testCluster := cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: true,
		UUID:       "uuid-0",
		Handlers:   handlers,
	})
	defer testCluster.Close()

	t.Run("no-hosts", func(t *testing.T) {
		errorCode = http.StatusOK
		_, err := NewClient(nil, "user", "password", nil)
		require.Error(t, err)
	})

	type testCase struct {
		errorCode int
		body      []byte
	}

	cases := []testCase{
		{
			errorCode: http.StatusUnauthorized,
			body:      []byte("user and password required"),
		},
		{
			errorCode: http.StatusForbidden,
			body:      []byte(`{"permissions":["cluster_admin[*]"], "message":"Forbidden"}`),
		},
		{
			errorCode: http.StatusBadRequest,
			body:      []byte(`{"permissions":["cluster_admin[*]"], "message":"Forbidden"}`),
		},
	}

	for _, tc := range cases {
		t.Run(strconv.Itoa(tc.errorCode), func(t *testing.T) {
			errorCode = tc.errorCode
			count = 0
			body = tc.body

			_, err := NewClient([]string{testCluster.URL()}, "user", "password", nil)
			require.Error(t, err)

			var bootstrapFailure *cbrest.BootstrapFailureError
			require.ErrorAs(t, err, &bootstrapFailure)

			if !slice.ContainsInt([]int{http.StatusUnauthorized, http.StatusForbidden}, errorCode) {
				return
			}

			var authError AuthError
			require.ErrorAs(t, err, &authError)
			require.Equal(t, errorCode == http.StatusUnauthorized, authError.Authentication)
		})
	}
}

func TestNewClient(t *testing.T) {
	handler := &TestHandler{
		ClusterUUID: "cluster_x",
		PoolsDefault: TestPoolsDefaultData{
			ClusterName: "grumpy",
			Nodes: []struct {
				Version string `json:"version"`
			}{
				{
					Version: "7.0.0-0000-enterprise",
				},
			},
			StorageTotals: TestStorageTotals{
				HDD: TestQuota{
					QuotaTotal: 500,
					Used:       300,
					UsedByData: 200,
				},
				RAM: TestQuota{
					QuotaTotal: 1024,
					QuotaUsed:  300,
				},
			},
		},
		Nodes: []TestNode{
			{
				NodeUUID:          "node-0",
				Hostname:          "127.0.0.1:9000",
				Services:          []string{"kv", "backup"},
				Version:           "7.0.0-0000-enterprise",
				Status:            "healthy",
				ClusterMembership: "active",
				Ports:             map[string]uint16{"httpsMgmt": 19000},
			},
		},
		NodesReturnCode: http.StatusOK,
	}

	handler.Start(t, false)
	defer handler.Close()

	client, err := NewClient([]string{handler.URL()}, "user", "password", nil)
	require.NoError(t, err)

	// make sure that the data is set correctly in the client
	require.NotZero(t, client.BootstrapTime)
	require.Equal(t, "cluster_x", client.ClusterInfo.ClusterUUID)
	require.Equal(t, "grumpy", client.ClusterInfo.ClusterName)

	expectedClusterInfo := &values.ClusterInfo{
		RAMQuota:       1024,
		RAMUsed:        300,
		DiskTotal:      500,
		DiskUsed:       300,
		DiskUsedByData: 200,
	}

	require.Equal(t, expectedClusterInfo, client.ClusterInfo.ClusterInfo)

	expectedNodeSummary := values.NodesSummary{
		{
			NodeUUID:          "node-0",
			Host:              "https://127.0.0.1:19000",
			Version:           "7.0.0-0000-enterprise",
			Status:            "healthy",
			ClusterMembership: "active",
			Services:          []string{"kv", "backup"},
		},
	}

	require.Equal(t, expectedNodeSummary, client.ClusterInfo.NodesSummary)
}
