// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/cbvalue"
)

type TestPoolsDefaultData struct {
	ClusterName   string            `json:"clusterName"`
	Status        string            `json:"status"`
	StorageTotals TestStorageTotals `json:"storageTotals"`
	Nodes         []struct {
		Version string `json:"version"`
	} `json:"nodes"`
}

type TestStorageTotals struct {
	HDD TestQuota `json:"hdd"`
	RAM TestQuota `json:"ram"`
}

type TestQuota struct {
	QuotaTotal uint64 `json:"quotaTotal"`
	QuotaUsed  uint64 `json:"quotaUsed"`
	Used       uint64 `json:"used"`
	UsedByData uint64 `json:"usedByData,omitempty"`
}

type SysStats struct {
	CPU       float64 `json:"cpu_utilization_rate"`
	SwapTotal uint64  `json:"swap_total"`
	SwapUsed  uint64  `json:"swap_used"`
}

type TestNode struct {
	NodeUUID           string            `json:"nodeUUID"`
	Hostname           string            `json:"hostname"`
	Services           []string          `json:"services"`
	Version            string            `json:"version"`
	Status             string            `json:"status"`
	ClusterMembership  string            `json:"clusterMembership"`
	Ports              map[string]uint16 `json:"ports"`
	AlternateAddresses *struct {
		External *AlternateAddresses `json:"external"`
	} `json:"alternate_addresses"`
	SystemStats SysStats        `json:"systemStats"`
	CPUCount    json.RawMessage `json:"cpuCount"`
}

type BucketsEndpointData struct {
	Name                   string `json:"name"`
	CompressionMode        string `json:"compressionMode"`
	ConflictResolutionType string `json:"conflictResolutionType"`
	BucketType             string `json:"bucketType"`
	StorageBackend         string `json:"storageBackend"`
	EvictionPolicy         string `json:"evictionPolicy"`
	NumReplicas            uint64 `json:"replicaNumber"`
	Quota                  struct {
		RAM uint64 `json:"ram"`
	} `json:"quota"`
	BasicStats struct {
		QuotaPercentUsed float64 `json:"quotaPercentUsed"`
		ItemCount        uint64  `json:"itemCount"`
	} `json:"basicStats"`
	Controllers struct {
		Flush string `json:"flush"`
	} `json:"controllers"`
	VBucketServerMap values.VBucketServerMap `json:"vBucketServerMap"`
}

type RemoteClustersEndpointData struct {
	ConnectivityStatus string `json:"connectivityStatus"`
	Hostname           string `json:"hostname"`
	Name               string `json:"name"`
}

type TestHandler struct {
	ClusterUUID            string
	PoolsDefault           TestPoolsDefaultData
	Nodes                  []TestNode
	NodesReturnCode        int
	NodesBytes             []byte
	AutoFailoverSettings   values.AutoFailoverSettings
	AutoFailoverReturnCode int
	UILogs                 values.UILogs
	SASLLogs               string
	LogName                string
	LogsReturnCode         int
	MetricsReturnCode      int
	Metrics                Metric
	NodeStorageCode        int
	NodeStorage            values.NodeStorage
	Buckets                []BucketsEndpointData
	RemoteClusters         []RemoteClustersEndpointData
	BucketReturnCode       int
	RemoteClustersCode     int

	Cluster *cbrest.TestCluster
}

func (h *TestHandler) Start(t *testing.T, https, enterprise bool) {
	handlers := make(cbrest.TestHandlers)

	// The cbrest client requires this to be set
	if len(h.PoolsDefault.Nodes) == 0 {
		h.PoolsDefault.Nodes = []struct {
			Version string `json:"version"`
		}{
			{Version: "7.0.0-4736-enterprise"},
		}
	}

	handlers.Add(http.MethodGet, string(PoolsNodesEndpoint), func(w http.ResponseWriter, r *http.Request) {
		wrapped := struct {
			Nodes []TestNode `json:"nodes"`
		}{Nodes: h.Nodes}

		marshalAndSendTestHelper(h.NodesReturnCode, &wrapped, h.NodesBytes, w)
	})

	handlers.Add(http.MethodGet, string(cbrest.EndpointPoolsDefault), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(http.StatusOK, &h.PoolsDefault, nil, w)
	})

	handlers.Add(http.MethodGet, string("/pools/default/terseClusterInfo"), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(http.StatusOK, nil, nil, w)
	})

	handlers.Add(http.MethodGet, string(AutoFailOverSettings), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(h.AutoFailoverReturnCode, &h.AutoFailoverSettings, []byte(`"some error"`), w)
	})

	handlers.Add(http.MethodGet, string(UILogsEndpoint),
		func(w http.ResponseWriter, r *http.Request) {
			marshalAndSendTestHelper(h.LogsReturnCode, &h.UILogs, []byte(`"some error"`), w)
		})

	handlers.Add(http.MethodGet, string(SASLLogsEndpoint.Format(h.LogName)), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(h.LogsReturnCode, &h.SASLLogs, []byte(`"some error"`), w)
	})

	handlers.Add(http.MethodGet, string(PoolsBucketEndpoint), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(h.BucketReturnCode, &h.Buckets, []byte{}, w)
	})

	handlers.Add(http.MethodGet, string(PoolsRemoteCluster), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(h.RemoteClustersCode, &h.RemoteClusters, []byte{}, w)
	})

	handlers.Add(http.MethodGet, string(PrometheusQueryEndpoint), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(h.MetricsReturnCode, &h.Metrics, []byte(`"some error"`), w)
	})

	handlers.Add(http.MethodGet, string(NodesSelfEndpoint), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(h.NodeStorageCode, &h.NodeStorage, []byte(`"some error`), w)
	})

	testNodes := make(cbrest.TestNodes, len(h.Nodes))
	for i, node := range h.Nodes {
		services := make([]cbrest.Service, len(node.Services))
		for i, service := range node.Services {
			services[i] = cbrest.Service(service)
		}

		testNodes[i] = &cbrest.TestNode{
			Version:    cbvalue.Version(node.Version),
			Status:     node.Status,
			Services:   services,
			AltAddress: node.AlternateAddresses != nil,
			SSL:        https,
		}
	}

	var tlsConfig *tls.Config
	if https {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	h.Cluster = cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: enterprise,
		UUID:       h.ClusterUUID,
		Nodes:      testNodes,
		Handlers:   handlers,
		TLSConfig:  tlsConfig,
	})
}

func (h *TestHandler) URL() string {
	return h.Cluster.URL()
}

func (h *TestHandler) Close() {
	if h.Cluster == nil {
		return
	}

	h.Cluster.Close()
	h.Cluster = nil
}

func marshalAndSendTestHelper(statusCode int, okData interface{}, errorBytes []byte, w http.ResponseWriter) {
	if statusCode == http.StatusOK {
		out, _ := json.Marshal(&okData)
		w.WriteHeader(statusCode)
		_, _ = w.Write(out)
		return
	}

	w.WriteHeader(statusCode)
	_, _ = w.Write(errorBytes)
}
