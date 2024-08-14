// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package statistics

import (
	"github.com/couchbase/tools-common/netutil"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

var (
	statusCluster = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "multimanager_cluster_checker_status",
		Help: "Checker results for cluster level checkers",
	}, []string{"cluster_uuid", "cluster_name", "name", "id"})
	statusNode = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "multimanager_node_checker_status",
		Help: "Checker results for node level checkers",
	}, []string{"cluster_uuid", "cluster_name", "node_uuid", "node_name", "name", "id"})
	statusBucket = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "multimanager_bucket_checker_status",
		Help: "Checker results for bucket level checkers",
	}, []string{"cluster_uuid", "cluster_name", "bucket", "name", "id"})
	statusErr = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "multimanager_checker_errored",
		Help: "Checkers which have failed to run",
	}, []string{"name", "cluster"})
)

// CheckStatus increments prometheus metrics based on checker result.
func CheckStatus(results []*values.WrappedCheckerResult, cluster *values.CouchbaseCluster) {
	for _, item := range results {
		if item.Result == nil {
			continue
		}

		if item.Error != nil {
			statusErr.WithLabelValues(item.Result.Name, item.Cluster).Inc()
			continue
		}

		id := values.AllCheckerDefs[item.Result.Name].ID

		switch {
		case item.Bucket != "":
			statusBucket.WithLabelValues(cluster.UUID, cluster.Name, item.Bucket,
				item.Result.Name, id).Set(float64(item.Result.Status.Int()))
		case item.Node != "":
			var node values.NodeSummary
			for _, test := range cluster.NodesSummary {
				if test.NodeUUID == item.Node {
					node = test
					break
				}
			}
			statusNode.WithLabelValues(cluster.UUID, cluster.Name, node.NodeUUID, netutil.TrimSchema(node.Host),
				item.Result.Name, id).Set(float64(item.Result.Status.Int()))
		default:
			statusCluster.WithLabelValues(cluster.UUID, cluster.Name,
				item.Result.Name, id).Set(float64(item.Result.Status.Int()))
		}
	}
}

// RegisterStatsCollection registers prometheus stats.
func RegisterStatsCollection() {
	prometheus.MustRegister(statusCluster, statusNode, statusBucket, statusErr)
}

// UnregisterStatsCollection unregisters prometheus stats.
func UnregisterStatsCollection() {
	prometheus.Unregister(statusCluster)
	prometheus.Unregister(statusNode)
	prometheus.Unregister(statusBucket)
	prometheus.Unregister(statusErr)
}
