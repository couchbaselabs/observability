// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

// Package discovery provides an interface for automated ways to discover Couchbase Server clusters.
package discovery

import (
	"context"
	"time"
)

//go:generate mockery --name CouchbaseClusterDiscovery
//go:generate mockery --name Manager

type CouchbaseClusterDiscovery interface {
	Discover(ctx context.Context) error
}

type Manager interface {
	Start(interval time.Duration)
	Stop()
	HasBeenDiscovered() chan ClusterDiscoveryStatus
}
