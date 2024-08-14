// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package status

import (
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

var allCheckerFns = map[string]values.CheckerFn{
	values.CheckOneServicePerNode:      oneServicePerNodeCheck,
	values.CheckSingleOrTwoNodeCluster: singleOrTwoNodeClusterCheck,
	values.CheckUnhealthyNode:          unhealthyNodesCheck,
	values.CheckMixedMode:              mixedModeCheck,
	values.CheckSupportedVersion:       supportedVersionCheck,
	"cacheRESTDataChecks":              cacheRESTDataChecks,
	"sharedClientChecks":               sharedClientChecks,
	"bucketEndpointCheckers":           bucketEndpointCheckers,
	"bucketStatCheckers":               bucketStatCheckers,
	"nonGABuildCheck":                  nonGABuildCheck,
	values.CheckActiveCluster:          activeClusterCheck,
	values.CheckNodeSwapUsage:          nodeSwapUsageCheck,
	values.CheckAsymmetricalCluster:    asymmetricalClusterCheck,
	"nodeSelfChecks":                   runNodeSelfChecks,
	"bucketMemcachedStatsCheckers":     bucketMemcachedStatsCheckers,
	"serviceCheck":                     checkNodeServiceStatus,
	values.CheckBelowMinMem:            belowMinMemCheck,
	values.CheckFreeMem:                freeMemCheck,
	values.CheckDuplicateNodeUUID:      checkDuplicateNodeUUIDs,
}
