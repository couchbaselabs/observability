// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package status

import (
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

// MonitorInterface will be used to allow switching the monitor in unit testing.
type MonitorInterface interface {
	Start(time.Duration)
	Stop()
	TriggerAPICheck(cluster *values.CouchbaseCluster) error
	GetProgressFor(uuid string) (*values.ClusterProgress, error)
}

// EmptyMonitor is the base to be used for testing monitors.
type EmptyMonitor struct{}

func (e *EmptyMonitor) Start(time.Duration)                                         {}
func (e *EmptyMonitor) Stop()                                                       {}
func (e *EmptyMonitor) TriggerAPICheck(cluster *values.CouchbaseCluster) error      { return nil }
func (e *EmptyMonitor) GetProgressFor(uuid string) (*values.ClusterProgress, error) { return nil, nil }
