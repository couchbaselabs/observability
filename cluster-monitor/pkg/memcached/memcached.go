// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package memcached

import (
	"fmt"

	"github.com/couchbase/tools-common/netutil"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/memcached/internal"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/cbrest"
)

// MemDClient wraps access to the memcached endpoints of a cluster.
type MemDClient struct {
	manager *internal.ClientManager
}

// NewMemcachedClient creates a new memcached client for a cluster.
func NewMemcachedClient(cluster *values.CouchbaseCluster) (*MemDClient, error) {
	restClient, err := couchbase.NewClient(cluster.NodesSummary.GetHosts(), cluster.User, cluster.Password,
		cluster.GetTLSConfig(), false)
	if err != nil {
		return nil, fmt.Errorf("could not create client to communicate with data nodes: %w", err)
	}

	kvHosts, err := restClient.GetAllServiceHosts(cbrest.ServiceData)
	if err != nil {
		return nil, fmt.Errorf("could not get data service hosts: %w", err)
	}
	// Normalize the hosts to avoid creating duplicate clients
	for i := range kvHosts {
		kvHosts[i] = netutil.TrimSchema(kvHosts[i])
	}

	return &MemDClient{
		manager: internal.NewClientManager(kvHosts, cluster.User, cluster.Password, cluster.GetTLSConfig()),
	}, nil
}

func (m *MemDClient) Hosts() []string {
	return m.manager.Hosts()
}

func (m *MemDClient) Close() error {
	return m.manager.Close()
}
