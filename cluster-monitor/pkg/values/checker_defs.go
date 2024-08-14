// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

const (
	CheckOneServicePerNode         = "oneServicePerNode"
	CheckSingleOrTwoNodeCluster    = "singleOrTwoNodeCluster"
	CheckUnhealthyNode             = "unhealthyNode"
	CheckMixedMode                 = "mixedMode"
	CheckServerQuota               = "serverQuota"
	CheckGlobalAutoCompaction      = "globalAutoCompaction"
	CheckAutoFailoverEnabled       = "autoFailoverEnabled"
	CheckMaxBuckets                = "maxBuckets"
	CheckMissingActiveVBuckets     = "missingActiveVBuckets"
	CheckMissingReplicaVBuckets    = "missingReplicaVBuckets"
	CheckDataLoss                  = "dataLoss"
	CheckSupportedVersion          = "supportedVersion"
	CheckResidentRatioTooLow       = "residentRatioTooLow"
	CheckNonGABuild                = "nonGABuild"
	CheckReplicaVBucketNumber      = "replicavBucketNumber"
	CheckActiveCluster             = "activeCluster"
	CheckBucketMemoryUsage         = "bucketMemoryUsage"
	CheckNodeSwapUsage             = "nodeSwapUsage"
	CheckAsymmetricalCluster       = "asymmetricalCluster"
	BucketCountChecks              = "bucketCountChecks"
	CheckNodeDiskSpace             = "nodeDiskSpace"
	CheckBackupLocation            = "backupLocationCheck"
	CheckBackupTaskOrphaned        = "backupTaskOrphaned"
	CheckBucketDCPPaused           = "bucketDCPPaused"
	CheckTHP                       = "THP"
	CheckServiceStatus             = "serviceStatus"
	CheckGSILogLevel               = "gsiLogLevel"
	CheckSharedFilesystems         = "sharedFilesystems"
	CheckLargeCheckpoints          = "largeCheckpoints"
	CheckIndexWithNoRedundancy     = "indexWithNoRedundancy"
	CheckBadRedundantIndex         = "badRedundantIndex"
	CheckTooManyIndexReplicas      = "tooManyIndexReplicas"
	CheckLongDCPStreamNames        = "longDcpStreamNames"
	CheckBelowMinMem               = "belowMinMem"
	CheckEmptyServerGroup          = "emptyGroup"
	CheckMemcachedFragmentation    = "memcachedFragmentation"
	CheckCushionManagedFail        = "cushionManagedFail"
	CheckUnknownStorageEngine      = "unknownStorageEngineCheck"
	CheckFreeMem                   = "freeMem"
	CheckProcessLimits             = "checkProcessLimits"
	CheckOOMKills                  = "oomKills"
	CheckN2NCommunication          = "n2nCommunication"
	CheckDuplicateNodeUUID         = "duplicateNodeUUID"
	CheckDeveloperPreview          = "developerPreviewCheck"
	CheckSupportedOS               = "supportedOS"
	CheckTooManySearchReplicas     = "tooManySearchReplicas"
	CheckMissingIndexPartitions    = "missingIndexPartitions"
	CheckImbalancedIndexPartitions = "imbalancedPartitions"
	CheckHistogramUnderflow        = "histogramUnderflow"
	CheckMaxTTLBucket              = "maxTTLBucket"
	CheckDefaultVBucketCount       = "defaultVBucketCount"
	CheckNodesForBucket            = "nodesForBucket"
	CheckPortStatus                = "checkPortStatus"
	CheckAutoFailoverForVM         = "autoFailoverForVM"
	CheckSharedStorage             = "possibleSharedStorage"
	CheckDocumentTooBig            = "documentTooBig"
	CheckHighPrometheusLoadTime    = "highPrometheusLoadTime"
	CheckAnalyticsJRE              = "analyticsJRE"
	CheckXdcrInvalidDatatype       = "xdcrInvalidDatatypeCheck"
)

var AllCheckerDefs map[string]CheckerDefinition

//go:embed checker_defs.yaml
var checkerDefsData []byte

func init() {
	if err := yaml.Unmarshal(checkerDefsData, &AllCheckerDefs); err != nil {
		panic(fmt.Errorf("failed to load checker definitions YAML: %w", err))
	}
}
