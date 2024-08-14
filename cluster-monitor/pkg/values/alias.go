// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

// ClusterAlias maps a cluster UUID to an arbitrary name. The idea is that this can be used in the REST API instead
// of the cluster UUID which makes the REST API friendlier. The reason to use this versus using the cluster name is that
// we cannot guarantee that cluster names are unique across all the clusters.
type ClusterAlias struct {
	Alias       string `json:"alias"`
	ClusterUUID string `json:"cluster_uuid"`
}
