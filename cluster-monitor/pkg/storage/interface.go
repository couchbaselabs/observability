// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package storage

import (
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

//go:generate mockery --name Store

type Store interface {
	IsInitialized() (bool, error)
	Close() error

	// cluster manager user functions
	AddUser(user *values.User) error
	GetUser(user string) (*values.User, error)

	// couchbase cluster management functions
	GetClusters(sensitive bool, enterpriseOnly bool) ([]*values.CouchbaseCluster, error)
	GetCluster(uuid string, sensitive bool) (*values.CouchbaseCluster, error)
	AddCluster(cluster *values.CouchbaseCluster) error
	DeleteCluster(uuid string) error
	UpdateCluster(cluster *values.CouchbaseCluster) error

	// manage cluster alias functions
	AddAlias(alias *values.ClusterAlias) error
	DeleteAlias(alias string) error
	GetAlias(alias string) (*values.ClusterAlias, error)

	// storing checker results
	SetCheckerResult(result *values.WrappedCheckerResult) error
	GetCheckerResult(search values.CheckerSearch) ([]*values.WrappedCheckerResult, error)
	DeleteCheckerResults(search values.CheckerSearch) error
	DeleteWhereNodesDoNotMatch(knownNodes []string, clusterUUID string) (int64, error)
	DeleteWhereBucketsDoNotMatch(knownBuckets []string, clusterUUID string) (int64, error)

	// dismissal related functions
	AddDismissal(dismissal values.Dismissal) error
	GetDismissals(search values.DismissalSearchSpace) ([]*values.Dismissal, error)
	DeleteDismissals(search values.DismissalSearchSpace) error
	DeleteExpiredDismissals() (int64, error)
	DeleteDismissalForUnknownBuckets(knownBuckets []string, clusterUUID string) (int64, error)
	DeleteDismissalForUnknownNodes(knownNodes []string, clusterUUID string) (int64, error)
	AddCloudCredentials(creds *values.Credential) error
	GetCloudCredentials(sensitive bool) ([]*values.Credential, error)
}
