// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package discovery

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

type ClusterDiscoveryStatus string

const (
	ClusterDiscoveryStatusFailure ClusterDiscoveryStatus = "DiscoveryFailure"
	ClusterDiscoveryStatusSuccess ClusterDiscoveryStatus = "DiscoverySuccess"
)

type ClusterDiscoveryManager struct {
	discovery CouchbaseClusterDiscovery
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	DiscoveryStatus chan ClusterDiscoveryStatus
}

// NewClusterDiscoveryManager is responsible for creating ClusterDiscoveryManager
// based on a specific CouchbaseClusterDiscovery (e.g. Prometheus)
func NewClusterDiscoveryManager(disco CouchbaseClusterDiscovery) (*ClusterDiscoveryManager, error) {
	dm := ClusterDiscoveryManager{
		discovery:       disco,
		DiscoveryStatus: make(chan ClusterDiscoveryStatus),
	}
	return &dm, nil
}

func (d *ClusterDiscoveryManager) Start(interval time.Duration) {
	zap.S().Infow("(Discovery Manager) Starting discovery", "interval", interval)
	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.wg.Add(1)
	go d.discoverLoop(interval)
}

// HasBeenDiscovered returns the channel which contains the cluster discovery info.
func (d *ClusterDiscoveryManager) HasBeenDiscovered() chan ClusterDiscoveryStatus {
	return d.DiscoveryStatus
}

func (d *ClusterDiscoveryManager) Stop() {
	if d.ctx == nil {
		return
	}
	zap.S().Info("(Discovery Manager) Stopping discovery")
	d.cancel()
	d.wg.Wait()
	d.ctx, d.cancel = nil, nil
}

func (d *ClusterDiscoveryManager) discoverLoop(interval time.Duration) {
	zap.S().Debug("(Discovery Manager) Performing initial discovery")
	if err := d.discovery.Discover(d.ctx); err != nil {
		zap.S().Errorw("(Discovery Manager) Error performing discovery", "err", err)
		d.DiscoveryStatus <- ClusterDiscoveryStatusFailure
	} else {
		zap.S().Infof("(Discovery Manager) Initial discovery done, next run will be in %v.", interval)
		d.DiscoveryStatus <- ClusterDiscoveryStatusSuccess
	}

	ticker := time.NewTicker(interval)
	defer func() {
		d.wg.Done()
		ticker.Stop()
	}()
	for {
		select {
		case <-ticker.C:
			zap.S().Debug("(Discovery Manager) Performing discovery")
			if err := d.discovery.Discover(d.ctx); err != nil {
				zap.S().Errorw("(Discovery Manager) Error performing discovery", "err", err)
				d.DiscoveryStatus <- ClusterDiscoveryStatusFailure
			} else {
				zap.S().Infof("(Discovery Manager) Discovery complete, next run will be in %v.", interval)
				d.DiscoveryStatus <- ClusterDiscoveryStatusSuccess
			}
		case <-d.ctx.Done():
			return
		}
	}
}
