// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package heart

import (
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

//go:generate mockery --name MonitorIFace

type MonitorIFace interface {
	Start(heartBeatFrequency time.Duration)
	Stop()
	HeartBeatCluster(cluster *values.CouchbaseCluster) error
}
