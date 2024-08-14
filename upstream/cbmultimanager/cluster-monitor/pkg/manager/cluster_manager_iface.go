// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import "github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

//go:generate mockery --name ClusterManager

type ClusterManager interface {
	Start() error
	Stop()
	UpdateClusterInfo(cluster *values.CouchbaseCluster)
	GetProgress() (*values.ClusterProgress, error)
	ManuallyRunCheckers() error
	ManuallyHeartBeat() error
}
