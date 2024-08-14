// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package internal

import (
	"crypto/tls"
	"fmt"
	"sync"
	"testing"
	"time"

	memcached "github.com/couchbase/gomemcached/client"
	"github.com/stretchr/testify/require"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/memcached/internal/mocks"
)

func TestClientCreationAndReuse(t *testing.T) {
	factory := new(mocks.McClientFactory)
	md := NewClientManager([]string{"N0", "N1"}, "", "", nil)
	md.factory = factory
	require.Len(t, md.clients, 0)

	factory.On("CreateClient", "N0", "", "", (*tls.Config)(nil)).
		Return(nil, nil).Once()

	err := md.ClientForNode("N0", func(_ *memcached.Client) error {
		return nil
	})
	require.NoError(t, err)
	require.Len(t, md.clients, 1)
	factory.AssertExpectations(t)

	// Now check that reusing a client doesn't create another
	err = md.ClientForNode("N0", func(_ *memcached.Client) error {
		return nil
	})
	require.NoError(t, err)
	require.Len(t, md.clients, 1)
	factory.AssertExpectations(t)

	factory.On("CreateClient", "N1", "", "", (*tls.Config)(nil)).
		Return(nil, nil).Once()

	err = md.ClientForNode("N1", func(_ *memcached.Client) error {
		return nil
	})
	require.NoError(t, err)
	require.Len(t, md.clients, 2)
	factory.AssertExpectations(t)

	err = md.ClientForNode("N1", func(_ *memcached.Client) error {
		return nil
	})
	require.NoError(t, err)
	require.Len(t, md.clients, 2)
	factory.AssertExpectations(t)

	// Cannot test md.Close() because we don't have a mock for memcached.Client
}

func TestClientCreationHandlesErrors(t *testing.T) {
	factory := new(mocks.McClientFactory)
	md := NewClientManager([]string{"N0", "N1"}, "", "", nil)
	md.factory = factory

	const testError = "TEST: client creation failed"
	factory.On("CreateClient", "N0", "", "", (*tls.Config)(nil)).
		Return(nil, fmt.Errorf(testError))

	err := md.ClientForNode("N0", func(_ *memcached.Client) error {
		return nil
	})
	require.Error(t, err, testError)
}

func TestClientForNodeConcurrency(t *testing.T) {
	factory := new(mocks.McClientFactory)
	md := NewClientManager([]string{"N0", "N1"}, "", "", nil)
	md.factory = factory
	factory.On("CreateClient", "N0", "", "", (*tls.Config)(nil)).Return(nil, nil)

	guard := newConcurrencyGuard(t, 1)
	wg := sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go md.ClientForNode("N0", func(_ *memcached.Client) error { //nolint:errcheck,unparam
			defer wg.Done()
			guard.Start()
			defer guard.Stop()
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}
	wg.Wait()
}
