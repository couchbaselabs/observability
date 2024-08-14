// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package prometheus

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"testing"

	"github.com/couchbase/tools-common/netutil"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/configuration"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	promMocks "github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/discovery/prometheus/mocks"
	storeMocks "github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage/mocks"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

var testConfig = configuration.Config{
	PrometheusBaseURL: "localhost:29000",
	PrometheusLabelSelector: map[string]string{
		"job": "couchbase-cluster",
	},
	CouchbaseUser:     "Administrator",
	CouchbasePassword: "password",
}

func init() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
}

func testServer(t *testing.T, clusterUUID string, is7 bool) (handler *couchbase.TestHandler, cbAddress string) {
	handler = &couchbase.TestHandler{
		PoolsDefault: couchbase.TestPoolsDefaultData{
			ClusterName:   "test-cluster",
			StorageTotals: couchbase.TestStorageTotals{},
			Nodes: []struct {
				Version string `json:"version"`
			}{},
		},
		ClusterUUID:      clusterUUID,
		BucketReturnCode: http.StatusOK,
		Buckets:          []couchbase.BucketsEndpointData{},
		NodesReturnCode:  http.StatusOK,
		Nodes:            []couchbase.TestNode{},
	}

	if is7 {
		handler.PoolsDefault.Nodes = append(handler.PoolsDefault.Nodes, struct {
			Version string `json:"version"`
		}{
			Version: "7.0.0-0000-enterprise",
		})
		handler.Nodes = append(handler.Nodes, couchbase.TestNode{
			NodeUUID: "node-0",
			Version:  "7.0.0-0000-enterprise",
		})
	} else {
		handler.PoolsDefault.Nodes = append(handler.PoolsDefault.Nodes, struct {
			Version string `json:"version"`
		}{
			Version: "6.6.3-9600-enterprise",
		})
		handler.Nodes = append(handler.Nodes, couchbase.TestNode{
			NodeUUID: "node-0",
			Version:  "6.6.3-9600-enterprise",
		})
	}

	handler.Start(t, false, true)

	noSchemaHost := netutil.TrimSchema(handler.URL())
	_, port, _ := net.SplitHostPort(noSchemaHost)
	portNum, err := strconv.Atoi(port)
	require.NoError(t, err, "invalid port")

	handler.Nodes[0].Hostname = noSchemaHost
	handler.Nodes[0].Ports = map[string]uint16{
		"httpMgmt": uint16(portNum),
	}

	cbAddress = handler.URL()

	return //nolint:nakedret
}

func TestDiscoverNoTargets(t *testing.T) {
	store := storeMocks.Store{}
	disco, err := NewPrometheusCouchbaseClusterDiscovery(&testConfig, &store)
	require.NoError(t, err)
	mockProm := promMocks.PromAPI{}
	disco.prom = &mockProm

	mockProm.On("Targets", mock.Anything).Return(v1.TargetsResult{
		Active: []v1.ActiveTarget{},
	}, nil)
	store.On("GetClusters", mock.Anything, mock.Anything).Return([]*values.CouchbaseCluster{}, nil)

	err = disco.Discover(context.Background())
	require.NoError(t, err)

	store.AssertNumberOfCalls(t, "GetCluster", 0)
	store.AssertNumberOfCalls(t, "AddCluster", 0)
}

func TestDiscoverLabelMismatch(t *testing.T) {
	store := storeMocks.Store{}
	disco, err := NewPrometheusCouchbaseClusterDiscovery(&testConfig, &store)
	require.NoError(t, err)
	mockProm := promMocks.PromAPI{}
	disco.prom = &mockProm

	mockProm.On("Targets", mock.Anything).Once().Return(v1.TargetsResult{
		Active: []v1.ActiveTarget{
			{
				DiscoveredLabels: map[string]string{
					AddressLabel: "INVALID",
				},
				Labels: map[model.LabelName]model.LabelValue{
					"missing": "label",
				},
			},
			{
				DiscoveredLabels: map[string]string{
					AddressLabel: "INVALID",
				},
				Labels: map[model.LabelName]model.LabelValue{
					"job": "skipped",
				},
			},
		},
	}, nil)
	store.On("GetClusters", mock.Anything, mock.Anything).Return([]*values.CouchbaseCluster{}, nil)

	err = disco.Discover(context.Background())
	require.NoError(t, err)

	store.AssertNumberOfCalls(t, "GetCluster", 0)
	store.AssertNumberOfCalls(t, "AddCluster", 0)
}

func TestDiscoverOneTarget(t *testing.T) {
	store := storeMocks.Store{}
	disco, err := NewPrometheusCouchbaseClusterDiscovery(&testConfig, &store)
	require.NoError(t, err)
	mockProm := promMocks.PromAPI{}
	disco.prom = &mockProm

	testHandler, cbAddress := testServer(t, "TDOT-0", true)
	defer testHandler.Close()

	mockProm.On("Targets", mock.Anything).Return(v1.TargetsResult{
		Active: []v1.ActiveTarget{
			{
				DiscoveredLabels: map[string]string{
					AddressLabel: netutil.TrimSchema(cbAddress),
				},
				Labels: map[model.LabelName]model.LabelValue{
					"job": "couchbase-cluster",
				},
			},
		},
	}, nil)

	store.On("GetCluster", "TDOT-0", mock.Anything).Return(nil, values.ErrNotFound)
	store.On("AddCluster", mock.Anything).Return(nil)
	store.On("GetClusters", mock.Anything, mock.Anything).Return([]*values.CouchbaseCluster{
		{
			UUID: "TDOT-0",
		},
	}, nil)

	err = disco.Discover(context.Background())
	require.NoError(t, err)

	store.AssertNumberOfCalls(t, "GetCluster", 1)
	store.AssertNumberOfCalls(t, "AddCluster", 1)
}

func TestDiscoverExistingTarget(t *testing.T) {
	store := storeMocks.Store{}
	disco, err := NewPrometheusCouchbaseClusterDiscovery(&testConfig, &store)
	require.NoError(t, err)
	mockProm := promMocks.PromAPI{}
	disco.prom = &mockProm

	testHandler, cbAddress := testServer(t, "TDET-0", true)
	defer testHandler.Close()

	mockProm.On("Targets", mock.Anything).Return(v1.TargetsResult{
		Active: []v1.ActiveTarget{
			{
				DiscoveredLabels: map[string]string{
					AddressLabel: netutil.TrimSchema(cbAddress),
				},
				Labels: map[model.LabelName]model.LabelValue{
					"job": "couchbase-cluster",
				},
			},
		},
	}, nil)

	store.On("GetCluster", "TDET-0", mock.Anything).Return(&values.CouchbaseCluster{
		UUID: "TDET-0",
	}, nil)
	store.On("GetClusters", mock.Anything, mock.Anything).Return([]*values.CouchbaseCluster{
		{
			UUID: "TDET-0",
		},
	}, nil)

	require.NoError(t, err)

	err = disco.Discover(context.Background())
	require.NoError(t, err)

	store.AssertNumberOfCalls(t, "GetCluster", 1)
	store.AssertNumberOfCalls(t, "AddCluster", 0)
	store.AssertNumberOfCalls(t, "GetClusters", 1)
}

func TestDiscoverMultipleTargetsSameCluster(t *testing.T) {
	store := storeMocks.Store{}
	disco, err := NewPrometheusCouchbaseClusterDiscovery(&testConfig, &store)
	require.NoError(t, err)
	mockProm := promMocks.PromAPI{}
	disco.prom = &mockProm

	testHandler, cbAddress := testServer(t, "TDMTSC-0", true)
	defer testHandler.Close()

	mockProm.On("Targets", mock.Anything).Return(v1.TargetsResult{
		Active: []v1.ActiveTarget{
			{
				DiscoveredLabels: map[string]string{
					AddressLabel: netutil.TrimSchema(cbAddress),
				},
				Labels: map[model.LabelName]model.LabelValue{
					"job": "couchbase-cluster",
				},
			},
			{
				DiscoveredLabels: map[string]string{
					AddressLabel: netutil.TrimSchema(cbAddress),
				},
				Labels: map[model.LabelName]model.LabelValue{
					"job": "couchbase-cluster",
				},
			},
		},
	}, nil)

	store.On("GetCluster", "TDMTSC-0", mock.Anything).Return(&values.CouchbaseCluster{
		UUID: "TDMTSC-0",
	}, nil)
	store.On("GetClusters", mock.Anything, mock.Anything).Return([]*values.CouchbaseCluster{
		{
			UUID: "TDMTSC-0",
		},
	}, nil)

	require.NoError(t, err)

	err = disco.Discover(context.Background())
	require.NoError(t, err)

	store.AssertNumberOfCalls(t, "GetCluster", 1)
	store.AssertNumberOfCalls(t, "AddCluster", 0)
	store.AssertNumberOfCalls(t, "GetClusters", 1)
}

func TestDiscoverGone(t *testing.T) {
	store := storeMocks.Store{}
	disco, err := NewPrometheusCouchbaseClusterDiscovery(&testConfig, &store)
	require.NoError(t, err)
	mockProm := promMocks.PromAPI{}
	disco.prom = &mockProm

	mockProm.On("Targets", mock.Anything).Once().Return(v1.TargetsResult{
		Active: []v1.ActiveTarget{},
	}, nil)
	store.On("GetCluster", "TDG-0", mock.Anything).Return(&values.CouchbaseCluster{
		UUID: "uuid-0",
	}, nil)
	store.On("GetClusters", mock.Anything, mock.Anything).Return([]*values.CouchbaseCluster{
		{
			UUID: "TDG-0",
		},
	}, nil)
	store.On("DeleteCluster", "TDG-0").Return(nil)

	err = disco.Discover(context.Background())
	require.NoError(t, err)

	store.AssertNumberOfCalls(t, "GetCluster", 0)
	store.AssertNumberOfCalls(t, "AddCluster", 0)
	store.AssertNumberOfCalls(t, "GetClusters", 1)
	store.AssertCalled(t, "DeleteCluster", "TDG-0")
}

// TODO: we don't have a good way of testing CB6 support without CMOS-91 - the cbrest test server starts on a
// random port, when we'd need it on either 8091 or 18091.
