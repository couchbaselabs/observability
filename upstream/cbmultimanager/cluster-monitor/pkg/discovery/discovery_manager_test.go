// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package discovery_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/discovery"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/discovery/mocks"
)

func TestDiscoveryLoop(t *testing.T) {
	t.Parallel()
	mockDisco := mocks.CouchbaseClusterDiscovery{}
	mockDisco.On("Discover", mock.Anything).Return(nil)
	dm, err := discovery.NewClusterDiscoveryManager(&mockDisco)
	require.NoError(t, err)
	dm.Start(time.Second)
	time.Sleep(1500 * time.Millisecond)
	status := <-dm.DiscoveryStatus
	assert.Equal(t, discovery.ClusterDiscoveryStatusSuccess, status)
	dm.Stop()
	mockDisco.AssertNumberOfCalls(t, "Discover", 1)
}

func TestDiscoveryLoopError(t *testing.T) {
	t.Parallel()
	mockDisco := mocks.CouchbaseClusterDiscovery{}
	mockDisco.On("Discover", mock.Anything).Return(fmt.Errorf("oh no"))
	dm, err := discovery.NewClusterDiscoveryManager(&mockDisco)
	require.NoError(t, err)
	dm.Start(time.Second)
	time.Sleep(1500 * time.Millisecond)
	status := <-dm.DiscoveryStatus
	assert.Equal(t, discovery.ClusterDiscoveryStatusFailure, status)
	dm.Stop()
	mockDisco.AssertNumberOfCalls(t, "Discover", 1)
}

func TestDiscoveryStopBeforeStart(t *testing.T) {
	t.Parallel()
	mockDisco := mocks.CouchbaseClusterDiscovery{}
	mockDisco.On("Discover", mock.Anything).Return(nil)
	dm, err := discovery.NewClusterDiscoveryManager(&mockDisco)
	require.NoError(t, err)
	dm.Stop()
	mockDisco.AssertNumberOfCalls(t, "Discover", 0)
}
