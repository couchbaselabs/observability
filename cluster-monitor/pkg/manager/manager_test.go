// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/configuration"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/discovery"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/discovery/mocks"
	storeMocks "github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage/mocks"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestManagerStartAndStop(t *testing.T) {
	testDir := t.TempDir()

	t.Run("doesn't start any cluster manager for NO Prom URL passed automatically", func(t *testing.T) {
		mockStore := new(storeMocks.Store)

		config := &configuration.Config{
			SQLiteKey:  "password",
			SQLiteDB:   filepath.Join(testDir, "db.sqlite"),
			MaxWorkers: 1,
		}

		m, err := NewManager(config)
		m.store = mockStore
		assert.Nil(t, err, "Expected to be able to create the manager")

		mockStore.On("GetClusters", mock.Anything, mock.Anything).Return([]*values.CouchbaseCluster{}, nil)

		go m.Start(DefaultFrequencyConfiguration)
		time.Sleep(1000 * time.Millisecond)
		m.Stop()

		mockStore.AssertNumberOfCalls(t, "GetClusters", 2)
	})

	t.Run("starts cluster manager for all discovered cb clusters Prom URL passed automatically", func(t *testing.T) {
		mockDiscoMgr := new(mocks.Manager)
		mockStore := new(storeMocks.Store)

		config := &configuration.Config{
			SQLiteKey:         "password",
			SQLiteDB:          filepath.Join(testDir, "db.sqlite"),
			MaxWorkers:        1,
			PrometheusBaseURL: "some-url",
			PrometheusLabelSelector: map[string]string{
				"job": "couchbase-cluster",
			},
		}

		m, err := NewManager(config)
		m.discoveryManager = mockDiscoMgr
		m.store = mockStore
		assert.Nil(t, err, "Expected to be able to create the manager")

		mockDiscoMgr.On("Start", mock.Anything)
		ch := make(chan discovery.ClusterDiscoveryStatus, 1)
		ch <- discovery.ClusterDiscoveryStatusFailure
		mockDiscoMgr.On("HasBeenDiscovered").Return(ch)

		mockDiscoMgr.On("Stop", mock.Anything)
		mockStore.On("GetClusters", mock.Anything, mock.Anything).Return([]*values.CouchbaseCluster{}, nil)

		go m.Start(DefaultFrequencyConfiguration)
		time.Sleep(1000 * time.Millisecond)
		m.Stop()

		mockDiscoMgr.AssertNumberOfCalls(t, "Start", 1)
		mockDiscoMgr.AssertNumberOfCalls(t, "HasBeenDiscovered", 2)
		mockDiscoMgr.AssertNumberOfCalls(t, "Stop", 1)
		mockStore.AssertNumberOfCalls(t, "GetClusters", 2)
	})
}

// TestManagerKeysSetup checks that when a manager is created and runs it will create the keys needed for creating JWTs.
func TestManagerKeysSetup(t *testing.T) {
	testDir := t.TempDir()

	config := &configuration.Config{
		SQLiteKey:  "password",
		SQLiteDB:   filepath.Join(testDir, "db.sqlite"),
		MaxWorkers: 1,
	}

	manager, err := NewManager(config)
	assert.Nil(t, err, "Expected to be able to create the manager")

	go manager.Start(DefaultFrequencyConfiguration)
	time.Sleep(200 * time.Millisecond)
	manager.Stop()

	assert.Len(t, config.EncryptKey, 32, "expected 32 byte key")
	assert.Len(t, config.SignKey, 64, "expected 64 byte key")
}
