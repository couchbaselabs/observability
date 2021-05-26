package manager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/heart/mocks"
	"github.com/couchbaselabs/cbmultimanager/status"
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type statusCheckerTestCase struct {
	name               string
	clusterUUID        string
	checkerName        string
	query              url.Values
	expectedStatusCode int
	expectedCluster    *resultCluster
}

func TestGetClusterStatusReport(t *testing.T) {
	mgr := createTestManager(t)
	loadTestData(t, mgr.store)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	time.Sleep(100 * time.Millisecond)

	cases := []statusCheckerTestCase{
		{
			name:               "notFound",
			clusterUUID:        "notFound",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "OK",
			clusterUUID:        "uuid-0",
			expectedStatusCode: http.StatusOK,
			expectedCluster: &resultCluster{
				UUID: "uuid-0",
				Name: "Cluster-0",
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
				StatusResults: []*values.WrappedCheckerResult{
					{
						Cluster: "uuid-0",
						Result: &values.CheckerResult{
							Name:   "checker-0",
							Status: values.GoodCheckerStatus,
							Time:   time.Time{}.UTC(),
						},
					},
					{
						Cluster: "uuid-0",
						Node:    "Node-1",
						Result: &values.CheckerResult{
							Name:   "checker-1",
							Status: values.WarnCheckerStatus,
							Time:   time.Time{}.UTC(),
						},
					},
					{
						Cluster: "uuid-0",
						Result: &values.CheckerResult{
							Name:   "checker-2",
							Status: values.AlertCheckerStatus,
							Time:   time.Time{}.UTC(),
						},
					},
					{
						Cluster: "uuid-0",
						Result: &values.CheckerResult{
							Name:   "checker-3",
							Status: values.InfoCheckerStatus,
							Time:   time.Time{}.UTC(),
						},
					},
				},
			},
		},
		{
			name:               "withDismissals",
			clusterUUID:        "uuid-1",
			expectedStatusCode: http.StatusOK,
			expectedCluster: &resultCluster{
				UUID: "uuid-1",
				Name: "Cluster-1",
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
				StatusResults: []*values.WrappedCheckerResult{},
				Dismissed:     1,
			},
		},
		{
			name:               "nodeFilter",
			query:              url.Values{"node": []string{"Node-1"}},
			clusterUUID:        "uuid-0",
			expectedStatusCode: http.StatusOK,
			expectedCluster: &resultCluster{
				UUID: "uuid-0",
				Name: "Cluster-0",
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
				StatusResults: []*values.WrappedCheckerResult{
					{
						Cluster: "uuid-0",
						Node:    "Node-1",
						Result: &values.CheckerResult{
							Name:   "checker-1",
							Status: values.WarnCheckerStatus,
							Time:   time.Time{}.UTC(),
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var query string
			if tc.query != nil {
				query = tc.query.Encode()
			}

			runStatusCheckerTest(t, fmt.Sprintf("http://localhost:%d/api/v1/clusters/%s/status?%s", mgr.config.HTTPPort,
				tc.clusterUUID, query), tc)
		})
	}
}

func TestGetClusterStatusCheckerResult(t *testing.T) {
	mgr := createTestManager(t)
	loadTestData(t, mgr.store)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	time.Sleep(100 * time.Millisecond)

	cases := []statusCheckerTestCase{
		{
			name:               "notFound",
			clusterUUID:        "notFound",
			checkerName:        "checker-0",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "OK",
			clusterUUID:        "uuid-0",
			checkerName:        "checker-0",
			expectedStatusCode: http.StatusOK,
			expectedCluster: &resultCluster{
				UUID: "uuid-0",
				Name: "Cluster-0",
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
				StatusResults: []*values.WrappedCheckerResult{
					{
						Cluster: "uuid-0",
						Result: &values.CheckerResult{
							Name:   "checker-0",
							Status: values.GoodCheckerStatus,
							Time:   time.Time{}.UTC(),
						},
					},
				},
			},
		},
		{
			name:               "getDismissed",
			clusterUUID:        "uuid-1",
			checkerName:        "checker-0",
			expectedStatusCode: http.StatusOK,
			expectedCluster: &resultCluster{
				UUID: "uuid-1",
				Name: "Cluster-1",
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
				StatusResults: []*values.WrappedCheckerResult{
					{
						Cluster: "uuid-1",
						Result: &values.CheckerResult{
							Name:   "checker-0",
							Status: values.AlertCheckerStatus,
							Time:   time.Time{}.UTC(),
						},
					},
				},
			},
		},
		{
			name:               "bucketFilterEmpty",
			query:              url.Values{"bucket": []string{"bucket-1"}},
			clusterUUID:        "uuid-0",
			checkerName:        "checker-0",
			expectedStatusCode: http.StatusOK,
			expectedCluster: &resultCluster{
				UUID: "uuid-0",
				Name: "Cluster-0",
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
				StatusResults: []*values.WrappedCheckerResult{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var query string
			if tc.query != nil {
				query = tc.query.Encode()
			}

			runStatusCheckerTest(t, fmt.Sprintf("http://localhost:%d/api/v1/clusters/%s/status/%s?%s",
				mgr.config.HTTPPort, tc.clusterUUID, tc.checkerName, query), tc)
		})
	}
}

func runStatusCheckerTest(t *testing.T, url string, tc statusCheckerTestCase) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	req.SetBasicAuth("user", "password")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, tc.expectedStatusCode, res.StatusCode)
	if tc.expectedStatusCode != http.StatusOK {
		return
	}

	var responseCluster resultCluster
	require.NoError(t, json.NewDecoder(res.Body).Decode(&responseCluster))
	tc.expectedCluster.LastUpdate = responseCluster.LastUpdate
	require.Equal(t, tc.expectedCluster, &responseCluster)
}

func TestGetStatusCheckerDefinitions(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	time.Sleep(100 * time.Millisecond)

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v1/checkers", mgr.config.HTTPPort),
		nil)
	require.NoError(t, err)

	req.SetBasicAuth("user", "password")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)

	// The number and name of checkers are subject to change so just make sure that some checkers are returned.
	var checkers map[string]values.CheckerDefinition
	require.NoError(t, json.NewDecoder(res.Body).Decode(&checkers))
	require.Greater(t, len(checkers), 0)
}

func TestGetStatusCheckerDefinition(t *testing.T) {
	mgr := createTestManager(t)

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	time.Sleep(100 * time.Millisecond)

	t.Run("notFound", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("http://localhost:%d/api/v1/checkers/notFoundChheckerName", mgr.config.HTTPPort), nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("OK", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v1/checkers/mixedMode",
			mgr.config.HTTPPort), nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusOK, res.StatusCode)

		// The number and name of checkers are subject to change so just make sure that some checkers are returned.
		var checker values.CheckerDefinition
		require.NoError(t, json.NewDecoder(res.Body).Decode(&checker))
		require.Equal(t, "mixedMode", checker.Name)
	})
}

type testMonitor struct {
	*status.EmptyMonitor
	triggerCount int
	err          error
}

func (t *testMonitor) TriggerAPICheck(cluster *values.CouchbaseCluster) error {
	t.triggerCount++
	return t.err
}

func TestTriggerAPIChecks(t *testing.T) {
	mgr := createTestManager(t)

	statusMonitor := &testMonitor{}
	mgr.statusMonitor = status.MonitorInterface(statusMonitor)

	t.Run("valid", func(t *testing.T) {
		require.HTTPSuccess(t, mgr.triggerAPIChecks, http.MethodPost, "/api/v1/statusChecks/api", nil)
		require.Equal(t, 1, statusMonitor.triggerCount, "expected status monitor to have been triggered")
	})

	t.Run("triggerError", func(t *testing.T) {
		statusMonitor.err = assert.AnError
		require.HTTPStatusCode(t, mgr.triggerAPIChecks, http.MethodPost, "/api/v1/statusChecks/api",
			nil, http.StatusInternalServerError)
	})
}

func TestRefreshCluster(t *testing.T) {
	mgr := createTestManager(t)

	var callNum int

	mockMonitor := new(mocks.MonitorIFace)
	mockMonitor.On("Start")
	mockMonitor.On("HeartBeatCluster", mock.Anything).Run(func(args mock.Arguments) {
		callNum++
	}).Return(nil)

	mgr.heartMonitor = mockMonitor

	statusMonitor := &testMonitor{}
	mgr.statusMonitor = status.MonitorInterface(statusMonitor)

	cluster := &values.CouchbaseCluster{
		UUID:     "uuid-0",
		Name:     "name-1",
		User:     "user",
		Password: "password",
		NodesSummary: values.NodesSummary{
			{
				NodeUUID: "N0",
				Host:     "https://localhost:9000",
			},
		},
	}

	require.NoError(t, mgr.store.AddCluster(cluster))

	mgr.setupKeys()
	mgr.startRESTServers()
	defer mgr.stopRESTServers()

	time.Sleep(100 * time.Millisecond)

	t.Run("404", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("http://localhost:%d/api/v1/clusters/uuid-4/refresh", testHTTPPort), nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		_ = res.Body.Close()

		require.Equal(t, http.StatusNotFound, res.StatusCode, "unexpected status code")
		time.Sleep(500 * time.Millisecond)
		require.Equal(t, 0, callNum)
	})

	t.Run("202", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("http://localhost:%d/api/v1/clusters/uuid-0/refresh", testHTTPPort), nil)
		require.NoError(t, err)

		req.SetBasicAuth("user", "password")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		_ = res.Body.Close()

		require.Equal(t, http.StatusOK, res.StatusCode, "unexpected status code")
		time.Sleep(500 * time.Millisecond)
		require.Equal(t, 1, callNum)
		require.Equal(t, 1, statusMonitor.triggerCount)
	})
}
