package status

import "github.com/couchbaselabs/cbmultimanager/values"

var AllCheckerDefs = map[string]values.CheckerDefinition{
	"oneServicePerNode": {
		Name:  "oneServicePerNode",
		Type:  values.APICheckerType,
		Title: "One Service Per Node",
		Description: "Checks that each node in the cluster is running only 1 service. Running multiple services" +
			" in one node can affect performance.",
	},
	"singleOrTwoNodeCluster": {
		Name:  "singleOrTwoNodeCluster",
		Type:  values.APICheckerType,
		Title: "Single or Two Node Cluster",
		Description: "Checks that production clusters have at least three nodes as clusters with fewer nodes " +
			"cannot use some features.",
	},
	"unhealthyNode": {
		Name:        "unhealthyNode",
		Type:        values.APICheckerType,
		Title:       "Unhealthy or Inactive Node",
		Description: "Checks that the node is active and healthy.",
	},
	"mixedMode": {
		Name:        "mixedMode",
		Type:        values.APICheckerType,
		Title:       "Mixed Version Cluster",
		Description: "Checks that production clusters have homogeneous versions.",
	},
	"serverQuota": {
		Name:        "serverQuota",
		Type:        values.APICheckerType,
		Title:       "Server Quota",
		Description: "Memory allocated to Couchbase Server should not be grater than 80% if system memory.",
	},
	"globalAutoCompaction": {
		Name:        "globalAutoCompaction",
		Type:        values.APICheckerType,
		Title:       "Auto Compaction Enabled",
		Description: "Production environments should have auto-compaction enabled to ensure disk space is reclaimed.",
	},
	"autoFailoverEnabled": {
		Name:        "autoFailoverEnabled",
		Type:        values.APICheckerType,
		Title:       "Auto-Failover Enabled",
		Description: "An autofailover timeout should be set to allow unresponsive nodes to failover automatically.",
	},
	"maxBuckets": {
		Name:        "maxBuckets",
		Type:        values.APICheckerType,
		Title:       "Number of Buckets",
		Description: "If the number of buckets is larger than 30 this can impact performance.",
	},
	"missingActiveVBuckets": {
		Name:        "missingActiveVBuckets",
		Type:        values.APICheckerType,
		Title:       "Missing Active vBuckets",
		Description: "Checks that the are no missing active vBuckets.",
	},
	"missingReplicaVBuckets": {
		Name:        "missingReplicaVBuckets",
		Type:        values.APICheckerType,
		Title:       "Missing Replica vBuckets",
		Description: "Checks that the are no missing replica vBuckets.",
	},
	"dataLoss": {
		Name:        "dataLoss",
		Type:        values.APICheckerType,
		Title:       "Data Loss Messages",
		Description: "Checks that no data loss messages have occurred.",
	},
	"supportedVersion": {
		Name:        "supportedVersion",
		Type:        values.APICheckerType,
		Title:       "Server Version Supportability",
		Description: "Checks that the Couchbase Server Version is within the support and maintenance periods.",
	},
	"residentRatioTooLow": {
		Name:        "residentRatioTooLow",
		Type:        values.APICheckerType,
		Title:       "Resident Ratio",
		Description: "Checks that the buckets Resident Ratio is more than 10%.",
	},
	"nonGABuild": {
		Name:        "nonGABuild",
		Type:        values.APICheckerType,
		Title:       "Generally Available Build",
		Description: "Checks that the build is recognised.",
	},
	"replicavBucketNumber": {
		Name:        "replicavBucketNumber",
		Type:        values.APICheckerType,
		Title:       "Number of Nodes for Replication",
		Description: "Checks the number of nodes is suitable for the number of replicas enabled.",
	},
	"activeCluster": {
		Name:        "activeCluster",
		Type:        values.APICheckerType,
		Title:       "All Nodes are Active",
		Description: "Checks that all nodes are available for a cluster.",
	},
	"bucketMemoryUsage": {
		Name:        "bucketMemoryUsage",
		Type:        values.APICheckerType,
		Title:       "Bucket Memory Usage",
		Description: "Checks that bucket memory usage is not above 95% of memory quota for more than 5 seconds.",
	},
	"nodeSwapUsage": {
		Name:        "nodeSwapUsage",
		Type:        values.APICheckerType,
		Title:       "Node Swap Usage",
		Description: "Checks that a node's system swap usage is not above 0.",
	},
	"asymmetricalCluster": {
		Name:        "asymmetricalCluster",
		Type:        values.APICheckerType,
		Title:       "Asymmetrical Cluster",
		Description: "Checks that nodes running the same service are equally provisioned.",
	},
	"cpuBucketCount": {
		Name:        "cpuBucketCount",
		Type:        values.APICheckerType,
		Title:       "CPU and Bucket Count",
		Description: "Checks that the amount of CPU's is greater than or matches the amount of buckets.",
	},
	"nodeDiskSpace": {
		Name:        "nodeDiskSpace",
		Type:        values.APICheckerType,
		Title:       "Disk Space Usage",
		Description: "Checks that disk usage per node is not at or above 90%.",
	},
	"backupLocationCheck": {
		Name:        "backupLocation",
		Type:        values.APICheckerType,
		Title:       "Backup Location Check",
		Description: "Checks that no node lost access to a backup repository in the last 3 days.",
	},
}

var allCheckerFns = map[string]values.CheckerFn{
	"oneServicePerNode":      oneServicePerNodeCheck,
	"singleOrTwoNodeCluster": singleOrTwoNodeClusterCheck,
	"unhealthyNode":          unhealthyNodesCheck,
	"mixedMode":              mixedModeCheck,
	"supportedVersion":       supportedVersionCheck,
	"sharedClientChecks":     sharedClientChecks,
	"bucketEndpointCheckers": bucketEndpointCheckers,
	"bucketStatCheckers":     bucketStatCheckers,
	"nonGABuildCheck":        nonGABuildCheck,
	"activeCluster":          activeClusterCheck,
	"nodeSwapUsage":          nodeSwapUsageCheck,
	"asymmetricalCluster":    asymmetricalClusterCheck,
	"cpuBucketCount":         cpuBucketCountCheck,
	"nodeDiskSpace":          runNodeSelfChecks,
}
