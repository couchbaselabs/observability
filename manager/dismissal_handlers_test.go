package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/storage"
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/stretchr/testify/require"
)

func loadDismissalTestData(t *testing.T, store storage.Store, loadDismissals bool) {
	// load clusters
	clusters := []values.CouchbaseCluster{
		{
			UUID:     "uuid-0",
			Name:     "Cluster-0",
			User:     "user",
			Password: "password",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID:          "Node-0",
					Version:           "7.0.0-0000-enterprise",
					Host:              "http://localhost:9000",
					ClusterMembership: "active",
					Status:            "status",
					Services:          []string{"kv"},
				},
			},
			BucketsSummary: values.BucketsSummary{
				{
					Name: "Bucket-0",
				},
			},
		},
		{
			UUID:     "uuid-1",
			Name:     "Cluster-1",
			User:     "user",
			Password: "password",
			NodesSummary: values.NodesSummary{
				{
					NodeUUID:          "Node-1",
					Version:           "7.0.0-0000-enterprise",
					Host:              "http://localhost:8091",
					ClusterMembership: "active",
					Status:            "status",
					Services:          []string{"kv"},
				},
			},
		},
	}

	for _, cluster := range clusters {
		require.NoError(t, store.AddCluster(&cluster))
	}

	if !loadDismissals {
		return
	}

	dismissals := []values.Dismissal{
		{
			ID:          "D0",
			Forever:     true,
			Level:       values.ClusterDismissLevel,
			ClusterUUID: "uuid-1",
			CheckerName: "checker-0",
		},
		{
			ID:          "D1",
			Forever:     true,
			Level:       values.AllDismissLevel,
			CheckerName: "checker-0",
		},
		{
			ID:          "D2",
			Forever:     true,
			Level:       values.BucketDismissLevel,
			ClusterUUID: "uuid-0",
			BucketName:  "Bucket-0",
			CheckerName: "checker-1",
		},
		{
			ID:          "D3",
			Forever:     true,
			Level:       values.NodeDismissLevel,
			ClusterUUID: "uuid-1",
			NodeUUID:    "Node-1",
			CheckerName: "checker-2",
		},
	}

	for _, dismissal := range dismissals {
		require.NoError(t, store.AddDismissal(dismissal))
	}
}

func TestDeleteDismissal(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	loadDismissalTestData(t, mgr.store, true)

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://localhost:%d/api/v1/dismissals/D0",
		mgr.config.HTTPPort), nil)
	require.NoError(t, err)

	req.SetBasicAuth("user", "password")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)

	dismissals, err := mgr.store.GetDismissals(values.DismissalSearchSpace{})
	require.NoError(t, err)
	require.Len(t, dismissals, 3)

	for _, dismissal := range dismissals {
		require.NotEqual(t, "D0", dismissal.ID)
	}
}

type dismissalTestCase struct {
	name        string
	filters     url.Values
	expectedIDs []string
}

func TestDeleteDismissals(t *testing.T) {
	cases := []dismissalTestCase{
		{
			name:        "byCluster",
			filters:     url.Values{"cluster": []string{"uuid-0"}},
			expectedIDs: []string{"D0", "D1", "D3"},
		},
		{
			name:        "byChecker",
			filters:     url.Values{"checker": []string{"checker-0"}},
			expectedIDs: []string{"D2", "D3"},
		},
		{
			name:        "byNode",
			filters:     url.Values{"node": []string{"Node-1"}},
			expectedIDs: []string{"D0", "D1", "D2"},
		},
		{
			name:        "byBucket",
			filters:     url.Values{"bucket": []string{"Bucket-0"}},
			expectedIDs: []string{"D0", "D1", "D3"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := createTestManager(t)

			mgr.setupKeys()
			mgr.startRESTServers()
			defer mgr.stopRESTServers()

			loadDismissalTestData(t, mgr.store, true)

			time.Sleep(100 * time.Millisecond)

			req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://localhost:%d/api/v1/dismissals?%s",
				mgr.config.HTTPPort, tc.filters.Encode()), nil)
			require.NoError(t, err)

			req.SetBasicAuth("user", "password")

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			_ = res.Body.Close()

			require.Equal(t, http.StatusOK, res.StatusCode)

			dismissals, err := mgr.store.GetDismissals(values.DismissalSearchSpace{})
			require.NoError(t, err)
			require.Len(t, dismissals, len(tc.expectedIDs))

			for _, dismissal := range dismissals {
				require.Contains(t, tc.expectedIDs, dismissal.ID)
			}
		})
	}
}

func TestGetDismissals(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	loadDismissalTestData(t, mgr.store, true)

	time.Sleep(100 * time.Millisecond)

	cases := []dismissalTestCase{
		{
			name:        "all",
			expectedIDs: []string{"D0", "D1", "D2", "D3"},
		},
		{
			name:        "byCluster",
			filters:     url.Values{"cluster": []string{"uuid-0"}},
			expectedIDs: []string{"D2"},
		},
		{
			name:        "byNode",
			filters:     url.Values{"node": []string{"Node-1"}, "cluster": []string{"uuid-1"}},
			expectedIDs: []string{"D3"},
		},
		{
			name:        "byBucket",
			filters:     url.Values{"bucket": []string{"Bucket-0"}, "cluster": []string{"uuid-0"}},
			expectedIDs: []string{"D2"},
		},
		{
			name:        "byChecker",
			filters:     url.Values{"checker": []string{"checker-0"}},
			expectedIDs: []string{"D0", "D1"},
		},
		{
			name:    "noMatch",
			filters: url.Values{"bucket": []string{"Bucket-0"}, "cluster": []string{"uuid-1"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v1/dismissals?%s",
				mgr.config.HTTPPort, tc.filters.Encode()), nil)
			require.NoError(t, err)

			req.SetBasicAuth("user", "password")

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			require.Equal(t, http.StatusOK, res.StatusCode)

			var dismissals []values.Dismissal
			require.NoError(t, json.NewDecoder(res.Body).Decode(&dismissals))
			require.Len(t, dismissals, len(tc.expectedIDs))

			for _, dismissal := range dismissals {
				require.Contains(t, tc.expectedIDs, dismissal.ID)
			}
		})
	}
}

func TestDismiss(t *testing.T) {
	type testCase struct {
		name              string
		request           []byte
		expectedStatus    int
		expectedDismissal *values.Dismissal
	}

	cases := []testCase{
		{
			name:           "invalidJSON",
			request:        []byte(`{"noClosing":1`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "noTimeConstraint",
			expectedStatus: http.StatusBadRequest,
			request:        []byte(`{"level":0,"checker_name":"mixedMode"}`),
		},
		{
			name:           "invalidDurationString",
			expectedStatus: http.StatusBadRequest,
			request:        []byte(`{"level":0,"checker_name":"mixedMode","dismiss_for":"77d32h44m7s"}`),
		},
		{
			name:           "noCheckerName",
			expectedStatus: http.StatusBadRequest,
			request:        []byte(`{"level":0,"forever":true}`),
		},
		{
			name:           "checkerNameDoesNotExists",
			expectedStatus: http.StatusBadRequest,
			request:        []byte(`{"level":0,"checker_name":"mixedMode333","forever":true}`),
		},
		{
			name:           "invalidAllDismissLevel",
			expectedStatus: http.StatusBadRequest,
			// level 0 cannot have cluster/bucket/node/log details
			request: []byte(`{"level":0,"checker_name":"mixedMode","forever":true,"cluster_uuid":"uuid-0"}`),
		},
		{
			name:           "validAllDismiss",
			expectedStatus: http.StatusOK,
			request:        []byte(`{"level":0,"checker_name":"mixedMode","forever":true}`),
			expectedDismissal: &values.Dismissal{
				Forever:     true,
				Level:       values.AllDismissLevel,
				CheckerName: "mixedMode",
			},
		},
		{
			name:           "invalidClusterDismissLevel",
			expectedStatus: http.StatusBadRequest,
			// a cluster uuid is required for dismissing at cluster level
			request: []byte(`{"level":1,"checker_name":"mixedMode","forever":true}`),
		},
		{
			name:           "clusterDoesNotExistClusterLevelDismiss",
			expectedStatus: http.StatusBadRequest,
			request:        []byte(`{"level":1,"checker_name":"mixedMode","forever":true,"cluster_uuid":"fake"}`),
		},
		{
			name:           "clusterLevelDismissExtraInfo",
			expectedStatus: http.StatusBadRequest,
			request: []byte(
				`{"level":1,"checker_name":"mixedMode","forever":true,"bucket_name":"b","cluster_uuid":"uuid-1"}`),
		},
		{
			name:           "clusterLevelDismissValid",
			expectedStatus: http.StatusOK,
			request:        []byte(`{"level":1,"checker_name":"mixedMode","forever":true,"cluster_uuid":"uuid-0"}`),
			expectedDismissal: &values.Dismissal{
				Forever:     true,
				Level:       values.ClusterDismissLevel,
				CheckerName: "mixedMode",
				ClusterUUID: "uuid-0",
			},
		},
		{
			name:           "bucketLevelDismissMissingBucket",
			expectedStatus: http.StatusBadRequest,
			request:        []byte(`{"level":2,"checker_name":"mixedMode","forever":true,"cluster_uuid":"uuid-0"}`),
		},
		{
			name:           "bucketLevelDismissMissingCluster",
			expectedStatus: http.StatusBadRequest,
			request:        []byte(`{"level":2,"checker_name":"mixedMode","forever":true,"bucket_name":"Bucket-0"}`),
		},
		{
			name:           "bucketLevelDismissBucketDoesNotExist",
			expectedStatus: http.StatusBadRequest,
			request: []byte(
				`{"level":2,"checker_name":"mixedMode","forever":true,"bucket_name":"B","cluster_uuid":"uuid-0"}`),
		},
		{
			name:           "validBucketLevelDismiss",
			expectedStatus: http.StatusOK,
			request: []byte(`{"level":2,"checker_name":"mixedMode","forever":true,"bucket_name":"Bucket-0",` +
				`"cluster_uuid":"uuid-0"}`),
			expectedDismissal: &values.Dismissal{
				Forever:     true,
				Level:       values.BucketDismissLevel,
				CheckerName: "mixedMode",
				ClusterUUID: "uuid-0",
				BucketName:  "Bucket-0",
			},
		},
		{
			name:           "nodeLevelDismissMissingNode",
			expectedStatus: http.StatusBadRequest,
			request:        []byte(`{"level":3,"checker_name":"mixedMode","forever":true,"cluster_uuid":"uuid-0"}`),
		},
		{
			name:           "nodeLevelDismissMissingCluster",
			expectedStatus: http.StatusBadRequest,
			request:        []byte(`{"level":3,"checker_name":"mixedMode","forever":true,"node_uuid":"Node-0"}`),
		},
		{
			name:           "nodeLevelDismissNodeDoesNotExist",
			expectedStatus: http.StatusBadRequest,
			request: []byte(
				`{"level":3,"checker_name":"mixedMode","forever":true,"node_uuid":"B","cluster_uuid":"uuid-0"}`),
		},
		{
			name:           "validNodeLevelDismiss",
			expectedStatus: http.StatusOK,
			request: []byte(`{"level":3,"checker_name":"mixedMode","forever":true,"node_uuid":"Node-0",` +
				`"cluster_uuid":"uuid-0"}`),
			expectedDismissal: &values.Dismissal{
				Forever:     true,
				Level:       values.NodeDismissLevel,
				CheckerName: "mixedMode",
				ClusterUUID: "uuid-0",
				NodeUUID:    "Node-0",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := createTestManager(t)

			mgr.setupKeys()
			mgr.startRESTServers()
			defer mgr.stopRESTServers()

			loadDismissalTestData(t, mgr.store, false)

			time.Sleep(100 * time.Millisecond)

			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d/api/v1/dismissals",
				mgr.config.HTTPPort), bytes.NewReader(tc.request))
			require.NoError(t, err)

			req.SetBasicAuth("user", "password")

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			require.Equal(t, tc.expectedStatus, res.StatusCode)

			if tc.expectedStatus != http.StatusOK {
				return
			}

			dismissals, err := mgr.store.GetDismissals(values.DismissalSearchSpace{})
			require.NoError(t, err)
			require.Len(t, dismissals, 1)

			tc.expectedDismissal.ID = dismissals[0].ID
			require.Equal(t, tc.expectedDismissal, dismissals[0])
		})
	}
}
