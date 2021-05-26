package couchbase

import (
	"github.com/couchbase/tools-common/cbrest"
)

const (
	PoolsNodesEndpoint       cbrest.Endpoint = "/pools/nodes"
	PoolsBucketEndpoint      cbrest.Endpoint = "/pools/default/buckets"
	PoolsBucketStatsEndpoint cbrest.Endpoint = "/pools/default/buckets/%s/stats"
	NodesSelfEndpoint        cbrest.Endpoint = "/nodes/self"

	UILogsEndpoint   cbrest.Endpoint = "/logs"
	SASLLogsEndpoint cbrest.Endpoint = "/sasl_logs/%s"

	AutoFailOverSettings cbrest.Endpoint = "/settings/autoFailover"

	PrometheusQueryEndpoint cbrest.Endpoint = "/_prometheus/api/v1/query_range"
)
