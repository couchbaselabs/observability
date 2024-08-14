// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package status

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/tools-common/cbvalue"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/memcached"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/netutil"
	"golang.org/x/exp/slices"
)

// bucketEndpointCheckers groups all the checkers that work on the data from the /pools/default/buckets endpoint as to
// avoid doing multiple REST calls.
func bucketEndpointCheckers(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	var dataNodes int
	for _, node := range cluster.NodesSummary {
		if slices.Contains(node.Services, "kv") {
			dataNodes++
		}
	}

	callTime := time.Now()
	buckets, err := cluster.GetCacheBuckets()
	if err != nil {
		return nil, fmt.Errorf("could not get pools/default/buckets: %w", err)
	}

	groupedChecks := []func(c *values.CouchbaseCluster, t time.Time) []*values.WrappedCheckerResult{
		missingActiveVBuckets,
		checkMB37643,
		checkDefaultVBucketCount,
		checkNodesForBucket,
		missingVBucketReplicas,
		unknownStorageEngineCheck,
	}

	results := replicavBucketNumber(buckets, callTime, cluster.UUID, dataNodes)

	for _, check := range groupedChecks {
		results = append(results, check(&cluster, callTime)...)
	}

	return results, nil
}

func bucketStatCheckers(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0)
	bucketsStats, errs := cluster.GetCacheBucketStats()

	for index, bucket := range cluster.BucketsSummary {
		callTime := time.Now()
		bucketStats := bucketsStats[index]

		if errs[bucket.Name] != nil {
			results = append(results, &values.WrappedCheckerResult{
				Cluster: cluster.UUID,
				Bucket:  bucket.Name,
				Error:   errs[bucket.Name],
			})
			continue
		}
		results = append(results, residentRatioTooLowCheck(bucketStats, bucket.Name, callTime, cluster.UUID))
		results = append(results, bucketMemoryUsageCheck(bucketStats, bucket.Name, callTime, cluster.UUID,
			bucket.Quota))
	}

	return results, nil
}

func bucketMemcachedStatsCheckers(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	callTime := time.Now()
	results := make([]*values.WrappedCheckerResult, 0)

	memClient, err := memcached.NewMemcachedClient(&cluster)
	if err != nil {
		return nil, fmt.Errorf("could not create memcached client for cluster: %w", err)
	}
	defer memClient.Close()

	for _, bucket := range cluster.BucketsSummary {
		dcpStatsResults, err := dcpStatsChecks(bucket.Name, callTime, &cluster, memClient)
		if err != nil {
			return nil, err
		}

		results = append(results, dcpStatsResults...)

		results = append(
			results,
			largeCheckpointsCheck(bucket.Name, callTime, &cluster, memClient),
			histogramUnderflowCheck(bucket.Name, callTime, &cluster, memClient),
		)

		fragStatsResults, err := memcachedFragCheck(bucket.Name, &cluster, memClient)
		if err != nil {
			return nil, err
		}
		results = append(results, fragStatsResults)
	}

	return results, nil
}

// histogramUnderflowCheck checks the stats of the given bucket for MB-40967.
func histogramUnderflowCheck(bucket string, callTime time.Time, cluster *values.CouchbaseCluster,
	client memcached.ConnIFace,
) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckHistogramUnderflow,
			Time:   callTime,
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
		Bucket:  bucket,
	}
	minVers := cluster.NodesSummary.GetMinVersion()
	if minVers.Older(cbvalue.Version6_5_0) || minVers.AtLeast("6.6.1") {
		return result
	}

	stats, err := client.DefaultStats(bucket)
	if err != nil {
		result.Result.Status = values.MissingCheckerStatus
		result.Error = fmt.Errorf("failed to acquire stats: %w", err)
		return result
	}

	for _, hostStats := range stats {
		for stat, value := range map[string]string{"GET": hostStats.CmdGet, "SET": hostStats.CmdSet} {
			if value == "" {
				result.Error = fmt.Errorf("got empty value for stat %s", stat)
				return result
			}
			val, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				// Should never happen
				result.Error = fmt.Errorf("invalid stat value for %s: %w", stat, err)
				return result
			}
			if val > math.MaxInt32 {
				result.Result.Status = values.WarnCheckerStatus
				result.Result.Remediation = fmt.Sprintf(
					"Timing statistics for %s operations may no longer be available. "+
						"Upgrade to Couchbase Server 6.6.1 or later. See MB-40967 for more details.", stat)
				return result
			}
			if val > uint64(math.Floor(math.MaxInt32*0.8)) {
				result.Result.Status = values.WarnCheckerStatus
				result.Result.Remediation = fmt.Sprintf(
					"Timing statistics for %s operations may soon no longer be available. "+
						"Upgrade to Couchbase Server 6.6.1 or later. See MB-40967 for more details.", stat)
				return result
			}
		}
	}

	// TODO(CMOS-228): this should be part of the "known issues" framework once that is developed
	result.Result.Status = values.InfoCheckerStatus
	result.Result.Remediation = "GET/SET timing statistics may become unavailable after 2.1B operations are " +
		"performed. Upgrade to Couchbase Server 6.6.1 or later. See MB-40967 for more details."
	return result
}

func maxBuckets(cluster *values.CouchbaseCluster, callTime time.Time) *values.WrappedCheckerResult {
	buckets, err := cluster.GetCacheBuckets()
	clusterUUID := cluster.UUID

	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckMaxBuckets,
			Time:   callTime,
			Status: values.GoodCheckerStatus,
		},
		Cluster: clusterUUID,
	}

	if err != nil {
		result.Error = err
		return result
	}

	result.Result.Value = []byte(fmt.Sprintf(`{"num_buckets":%d}`, len(buckets)))

	if len(buckets) > 30 {
		result.Result.Status = values.InfoCheckerStatus
		result.Result.Remediation = "Reduce the number of buckets as current number could cause performance " +
			"degradation."
	}

	return result
}

func missingActiveVBuckets(cluster *values.CouchbaseCluster, callTime time.Time,
) []*values.WrappedCheckerResult {
	buckets, err := cluster.GetCacheBuckets()
	clusterUUID := cluster.UUID

	results := make([]*values.WrappedCheckerResult, 0, len(buckets))

	// if we could not retrieve the buckets data, we return early and do not run the checker
	if err != nil {
		results = append(results, &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckMissingActiveVBuckets,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: clusterUUID,
			Error:   err,
		})
		return results
	}

	for _, bucket := range buckets {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckMissingActiveVBuckets,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: clusterUUID,
			Bucket:  bucket.Name,
		}

		missingActives := make([]int, 0)
		for vbid, vBucketStates := range bucket.VBucketServerMap.VBucketMap {
			if len(vBucketStates) == 0 || vBucketStates[0] == -1 {
				missingActives = append(missingActives, vbid)
			}
		}

		var err error
		result.Result.Value, err = json.Marshal(missingActives)
		if err != nil {
			result.Error = fmt.Errorf("could not marshall missing active slice: %w", err)
		}

		if len(missingActives) != 0 {
			result.Result.Status = values.AlertCheckerStatus
			result.Result.Remediation = "Some active vBuckets are missing, this could cause data loss. Please " +
				"rebalance the nodes in or add new ones."
		}

		results = append(results, result)
	}

	return results
}

func missingVBucketReplicas(cluster *values.CouchbaseCluster, callTime time.Time,
) []*values.WrappedCheckerResult {
	buckets, err := cluster.GetCacheBuckets()
	clusterUUID := cluster.UUID

	results := make([]*values.WrappedCheckerResult, 0, len(buckets))

	if err != nil {
		results = append(results, &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckMissingActiveVBuckets,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: clusterUUID,
			Error:   err,
		})
		return results
	}

	for _, bucket := range buckets {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckMissingReplicaVBuckets,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: clusterUUID,
			Bucket:  bucket.Name,
		}

		missingReplicas := make([]int, 0)
		// if there are supposed to be replicas check if any missing
		if bucket.VBucketServerMap.NumReplicas != 0 {
			for vbid, vBucketStates := range bucket.VBucketServerMap.VBucketMap {
				for i := 1; i < len(vBucketStates); i++ {
					if vBucketStates[i] == -1 {
						missingReplicas = append(missingReplicas, vbid)
					}
				}
			}
		}

		var err error
		result.Result.Value, err = json.Marshal(missingReplicas)
		if err != nil {
			result.Error = fmt.Errorf("could not marshal missing replicas slice: %w", err)
		}

		if len(missingReplicas) != 0 {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = "Some replica vBuckets are missing. Please rebalance the nodes in or add new" +
				" ones."
		}

		results = append(results, result)
	}

	return results
}

func residentRatioTooLowCheck(bucketStat *values.BucketStat, bucket string, callTime time.Time,
	clusterUUID string,
) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckResidentRatioTooLow,
			Status: values.GoodCheckerStatus,
			Time:   callTime,
		},
		Cluster: clusterUUID,
		Bucket:  bucket,
	}

	if len(bucketStat.VbActiveRatio) == 0 {
		result.Result.Status = values.MissingCheckerStatus
		result.Result.Value = []byte(`"Missing residency ratio value from REST endpoint. Will re-run soon."`)
		return result
	}

	resident := bucketStat.VbActiveRatio[len(bucketStat.VbActiveRatio)-1]
	if resident <= 5 {
		result.Result.Status = values.AlertCheckerStatus
		result.Result.Remediation = "Active Resident Ratio for this bucket is below 5%%" +
			"please increase Data Service Quota or allocate more memory"
	} else if resident <= 10 {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = "Active Resident Ratio for this bucket is less than 10%%" +
			"please increase Data Service Quota."
	}

	result.Result.Value = []byte(fmt.Sprintf(`{"residency": %.2f}`, resident))
	return result
}

func replicavBucketNumber(buckets []values.Bucket, callTime time.Time, clusterUUID string,
	nodeCount int,
) []*values.WrappedCheckerResult {
	results := make([]*values.WrappedCheckerResult, 0, len(buckets))
	for _, bucket := range buckets {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckReplicaVBucketNumber,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: clusterUUID,
			Bucket:  bucket.Name,
		}

		if bucket.VBucketServerMap.NumReplicas >= 3 && nodeCount < 10 {
			result.Result.Status = values.InfoCheckerStatus
			result.Result.Remediation = fmt.Sprintf("Data nodes required: %d is not suitable for number of replicas:"+
				" %d. 10 or more nodes recommended for 3 replicas.", bucket.VBucketServerMap.NumReplicas, nodeCount)
		} else if bucket.VBucketServerMap.NumReplicas >= 2 && nodeCount < 5 {
			result.Result.Status = values.InfoCheckerStatus
			result.Result.Remediation = fmt.Sprintf("Data nodes required: %d is not suitable for number of replicas:"+
				" %d. 5 or more nodes recommended for 2 replicas.", bucket.VBucketServerMap.NumReplicas, nodeCount)
		}

		results = append(results, result)
	}

	return results
}

func bucketMemoryUsageCheck(bucketStat *values.BucketStat, bucket string, callTime time.Time,
	clusterUUID string, quota uint64,
) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckBucketMemoryUsage,
			Status: values.GoodCheckerStatus,
			Time:   callTime,
		},
		Cluster: clusterUUID,
		Bucket:  bucket,
	}

	if len(bucketStat.MemUsed) < 5 {
		result.Result.Status = values.MissingCheckerStatus
		result.Result.Remediation = "Unable to check memory usage, will re-run soon."
		return result
	}

	var notAllAboveThreshold bool
	for _, memory := range bucketStat.MemUsed[len(bucketStat.MemUsed)-5 : len(bucketStat.MemUsed)] {
		if memory <= (float64(quota) * 0.95) {
			notAllAboveThreshold = true
			break
		}
	}

	if !notAllAboveThreshold {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = fmt.Sprintf("Memory used by bucket %s, has been above 95%% of the memory"+
			" quota for 5 seconds or more. Increase memory quota for bucket.", bucket)
	}

	return result
}

const (
	memcachedFragWarnPercentSix    = 15.0
	memcachedFragAlertPercentSix   = 20.0
	memcachedFragWarnPercentSeven  = 20.0
	memcachedFragAlertPercentSeven = 25.0
)

// memcachedFragCheck checks if too much of the memcached heap is fragmented
func memcachedFragCheck(bucket string, cluster *values.CouchbaseCluster,
	memClient memcached.ConnIFace,
) (*values.WrappedCheckerResult, error) {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckMemcachedFragmentation,
			Time:   cluster.LastUpdate,
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
		Bucket:  bucket,
	}
	var fragPercent float64
	var severity values.CheckerStatus
	memcachedFrag := make(map[string]string)
	remediation := "Too much of memcached's heap is fragmented. Contact Technical Support for details."
	memoryStats, err := memClient.MemStats(bucket)
	if err != nil {
		return nil, fmt.Errorf("could not collect memcached memory statistics: %w", err)
	}
	hostVersions, err := cluster.NodesSummary.GetHostVersions()
	if err != nil {
		return nil, fmt.Errorf("could not parse NodeSummary hostnames and versions, error: %w", err)
	}

	for _, memory := range memoryStats {
		// MemStats gets the hostname through the memcached client but NodeSummary gets
		// it through a normal client meaning the ports in the hostname will be different
		// This function splits the host and the port in the hostname so they will be the same.
		host, _, err := net.SplitHostPort(netutil.TrimSchema(memory.Host))
		if err != nil {
			return nil, fmt.Errorf("could not split hostname and port")
		}
		// Check version every time in case of mixed version cluster
		if hostVersions[host].Older(cbvalue.Version7_0_0) {
			fragPercent, severity, err = fragCalculator(memory.FragmentationBytes, memory.HeapBytes,
				memcachedFragWarnPercentSix, memcachedFragAlertPercentSix)
		} else {
			fragPercent, severity, err = fragCalculator(memory.ArenaFragmentationBytes, memory.ArenaResidentBytes,
				memcachedFragWarnPercentSeven, memcachedFragAlertPercentSeven)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to calculate fragmentation percent: %w", err)
		}

		if severity == values.AlertCheckerStatus {
			result.Result.Status = values.AlertCheckerStatus
		} else if severity == values.WarnCheckerStatus && result.Result.Status != values.AlertCheckerStatus {
			result.Result.Status = values.WarnCheckerStatus
		}

		memcachedFrag[memory.Host] = fmt.Sprintf("%.2f%%", fragPercent)
	}

	if result.Result.Status == values.AlertCheckerStatus || result.Result.Status == values.WarnCheckerStatus {
		result.Result.Remediation = remediation
	}

	result.Result.Value, _ = json.Marshal(memcachedFrag)
	return result, nil
}

func dcpStatsChecks(bucket string, callTime time.Time, cluster *values.CouchbaseCluster,
	memClient memcached.ConnIFace,
) ([]*values.WrappedCheckerResult, error) {
	var results []*values.WrappedCheckerResult

	dcpStats, err := memClient.DCPStats(bucket)
	if err != nil {
		return nil, fmt.Errorf("could not collect dcp statistics: %w", err)
	}

	results = append(
		results,
		checkMB46482(dcpStats, cluster, bucket, memClient, callTime),
		checkMB34280(dcpStats, cluster, bucket, callTime),
	)

	return results, nil
}

const maxDcpStreamNameLength = 200

func checkMB34280(stats []*memcached.DCPMemStats, cluster *values.CouchbaseCluster, bucket string,
	callTime time.Time,
) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckLongDCPStreamNames,
			Status: values.GoodCheckerStatus,
			Time:   callTime,
		},
		Cluster: cluster.UUID,
		Bucket:  bucket,
	}

	var breached []string
	for _, hostStats := range stats {
		for _, name := range hostStats.StreamNames {
			if len(name) > maxDcpStreamNameLength {
				breached = append(breached, name)
			}
		}
	}

	result.Result.Value, result.Error = json.Marshal(&breached)
	if result.Error != nil {
		return result
	}

	if len(breached) > 0 {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = fmt.Sprintf(
			"DCP stream names longer than %d characters will be forbidden in Couchbase"+
				" Server 7.0 and may cause issues in lower versions. See MB-34280 for more details. "+
				"Contact Couchbase Technical Support for advice.",
			maxDcpStreamNameLength)
	}
	return result
}

func checkMB46482(dcpStats []*memcached.DCPMemStats, cluster *values.CouchbaseCluster,
	bucket string, memClient memcached.ConnIFace, callTime time.Time,
) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckBucketDCPPaused,
			Status: values.GoodCheckerStatus,
			Time:   callTime,
		},
		Cluster: cluster.UUID,
		Bucket:  bucket,
	}

	// We can't simply set MinVersion/MaxVersion on the checker, as it's run by a "meta-checker" function, and those
	// are only applied at the top level.
	minVersion := cluster.NodesSummary.GetMinVersion()
	if minVersion.Older(cbvalue.Version6_5_0) || minVersion.AtLeast("6.6.3") {
		return result
	}

	vbStats, err := memClient.DefaultStats(bucket)
	if err != nil {
		result.Error = fmt.Errorf("could not collect VbActiveSync statistic: %w", err)
		return result
	}

	var nonZeroActiveSyncAccepted []*memcached.DefStats
	for _, hostStats := range vbStats {
		if hostStats.VbActiveSyncAccepted != "0" {
			nonZeroActiveSyncAccepted = append(nonZeroActiveSyncAccepted, hostStats)
		}
	}
	for _, dcp := range dcpStats {
		for _, badVb := range nonZeroActiveSyncAccepted {
			if dcp.Host != badVb.Host {
				continue
			}

			if dcp.MaxBufferBytes == nil {
				continue
			}

			if badVb.VbActiveSyncAccepted >= dcp.MaxBufferBytes[0].Value && dcp.PausedReason != nil {
				if dcp.PausedReason[0].Extras == "PausedReason::BufferLogFull" {
					result.Result.Status = values.AlertCheckerStatus
					result.Result.Remediation = fmt.Sprintf("DCP buffer may be full, see MB-46482."+
						"Writes may be paused for host %s. Contact support for analysis.", dcp.Host)
					break
				}

				result.Result.Status = values.WarnCheckerStatus
				result.Result.Remediation = fmt.Sprintf("Accepted replications higher than max dcp buffer"+
					" for host %s - you may be encountering MB-46482.", dcp.Host)
			}
		}
	}
	return result
}

const memUsageStat = "mem_usage"

// Threshold size of checkpoints per Bucket (Bytes)
const checkpointSizeMinimum = 50 * 1_000_000

// Threshold size of checkpoints as a percentage of the bucket quota
const checkpointSizeMinimumPct = 1

func largeCheckpointsCheck(bucket string, callTime time.Time, cluster *values.CouchbaseCluster,
	memClient memcached.ConnIFace,
) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckLargeCheckpoints,
			Status: values.GoodCheckerStatus,
			Time:   callTime,
		},
		Cluster: cluster.UUID,
		Bucket:  bucket,
	}

	bucketInfo := cluster.BucketsSummary.GetBucket(bucket)
	if bucketInfo == nil {
		result.Result.Status = values.MissingCheckerStatus
		result.Error = fmt.Errorf("could not get bucket information for bucket %s", bucket)
		return result
	}

	// Run this check for every host with this bucket, just running it against one is not sufficient
	for _, host := range memClient.Hosts() {
		host := netutil.TrimSchema(host)
		stats, err := memClient.CheckpointStats(host, bucket)
		if err != nil {
			result.Result.Status = values.MissingCheckerStatus
			result.Error = fmt.Errorf("failed to collect checkpoint stats for bucket %s on host %s: %w", bucket, host, err)
			return result
		}
		var largestVBID int
		var largestVBCheckpointSize int64
		for vb, stats := range stats {
			if sizeStr, ok := stats[memUsageStat]; ok {
				size, err := strconv.ParseInt(sizeStr, 10, 64)
				if err != nil {
					result.Result.Status = values.MissingCheckerStatus
					result.Error = fmt.Errorf("failed to parse checkpoint stats for bucket %s on host %s: %w", bucket, host, err)
					return result
				}
				if size > largestVBCheckpointSize {
					largestVBID = vb
					largestVBCheckpointSize = size
				}
			}
		}

		largestVbCheckpointSizePct := ((float64(largestVBCheckpointSize) / float64(bucketInfo.Quota)) * 100)
		if largestVBCheckpointSize > checkpointSizeMinimum && largestVbCheckpointSizePct > checkpointSizeMinimumPct {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = fmt.Sprintf("Bucket %s has large checkpoints (largest vBucket %d "+
				"on node %s, checkpoint size %dB, %.1f%% of bucket quota). Contact Technical Support for analysis.",
				bucket, largestVBID, host, largestVBCheckpointSize, largestVbCheckpointSizePct,
			)
		}
	}
	return result
}

func unknownStorageEngineCheck(cluster *values.CouchbaseCluster, callTime time.Time,
) []*values.WrappedCheckerResult {
	buckets, err := cluster.GetCacheBuckets()
	clusterUUID := cluster.UUID

	results := make([]*values.WrappedCheckerResult, 0, len(buckets))

	if err != nil {
		results = append(results, &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckMissingActiveVBuckets,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: clusterUUID,
			Error:   err,
		})
		return results
	}

	for _, bucket := range buckets {
		// Only check where this stat exists (versions >7.0)
		if bucket.StorageEngine == "" {
			continue
		}
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckUnknownStorageEngine,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: clusterUUID,
			Bucket:  bucket.Name,
		}

		switch bucket.StorageEngine {
		case "couchstore", "ephemeral", "magma":
		default:
			result.Result.Status = values.AlertCheckerStatus
			result.Result.Remediation = "Contact Couchbase Technical Support for analysis."
			result.Result.Value, _ = json.Marshal(&bucket.StorageEngine)
		}
		results = append(results, result)
	}
	return results
}

const thirtyDaysInSeconds = 30 * 24 * 60 * 60

func checkMB37643(cluster *values.CouchbaseCluster,
	callTime time.Time,
) []*values.WrappedCheckerResult {
	buckets, err := cluster.GetCacheBuckets()
	results := make([]*values.WrappedCheckerResult, 0, len(buckets))

	if err != nil {
		results = append(results, &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckMissingActiveVBuckets,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: cluster.UUID,
			Error:   err,
		})
		return results
	}

	isVulnerableVersion := false
	minVersion := cluster.NodesSummary.GetMinVersion()
	uptime, err := cluster.NodesSummary.GetHighestUptime()

	if minVersion.AtLeast(cbvalue.Version5_5_0) && minVersion.Older(cbvalue.Version("6.0.4")) {
		isVulnerableVersion = true
	}

	for _, bucket := range buckets {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckMaxTTLBucket,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: cluster.UUID,
			Bucket:  bucket.Name,
		}

		if isVulnerableVersion && bucket.MaxTTL >= thirtyDaysInSeconds {
			if uptime >= bucket.MaxTTL {
				result.Result.Remediation = "Bucket has hit MB-37643 - "
				result.Result.Status = values.AlertCheckerStatus
			} else {
				hitTime := time.Now().Add(time.Second * time.Duration(bucket.MaxTTL-uptime))
				result.Result.Remediation = fmt.Sprintf("Bucket will hit MB-37643 by %s time -- ", hitTime.Format(time.RFC1123))
				result.Result.Status = values.WarnCheckerStatus
			}
			result.Result.Remediation += fmt.Sprint("Can be resolved by using absolute time " +
				"instead of time from bucket creation in current version. Resolved in 6.0.4 and higher.")
		}

		if err != nil {
			result.Error = err
		}
		results = append(results, result)
	}
	return results
}

const (
	defaultVBucketCount = 1024
	macOSVBucketCount   = 64
)

func checkDefaultVBucketCount(cluster *values.CouchbaseCluster,
	callTime time.Time,
) []*values.WrappedCheckerResult {
	buckets, err := cluster.GetCacheBuckets()
	results := make([]*values.WrappedCheckerResult, 0, len(buckets))

	if err != nil {
		results = append(results, &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckMissingActiveVBuckets,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: cluster.UUID,
			Error:   err,
		})
		return results
	}

	macOS := false
	for _, node := range cluster.NodesSummary {
		if strings.Contains(node.OS, "-apple-") {
			macOS = true
		}
	}

	for _, bucket := range buckets {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckDefaultVBucketCount,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: cluster.UUID,
			Bucket:  bucket.Name,
		}

		vBucketsCount := len(bucket.VBucketServerMap.VBucketMap)

		if !macOS && vBucketsCount != defaultVBucketCount {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = fmt.Sprintf("Switch to default vBucket count: %v", defaultVBucketCount)
		} else if macOS && vBucketsCount != macOSVBucketCount {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = fmt.Sprintf("Switch to default vBucket count: %v", macOSVBucketCount)
		}

		results = append(results, result)
	}
	return results
}

func checkNodesForBucket(cluster *values.CouchbaseCluster,
	callTime time.Time,
) []*values.WrappedCheckerResult {
	buckets, err := cluster.GetCacheBuckets()
	results := make([]*values.WrappedCheckerResult, 0, len(buckets))

	if err != nil {
		results = append(results, &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckMissingActiveVBuckets,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: cluster.UUID,
			Error:   err,
		})
		return results
	}

	totalKVNodeCount := 0

	for _, node := range cluster.NodesSummary {
		if node.HasService("kv") {
			totalKVNodeCount++
		}
	}

	for _, bucket := range buckets {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckNodesForBucket,
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: cluster.UUID,
			Bucket:  bucket.Name,
		}

		if totalKVNodeCount > len(bucket.VBucketServerMap.ServerList) {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = "Verify all nodes are online, and rebalance if necessary." +
				" If this still persists, contact Couchbase Technical Support."
		}

		results = append(results, result)
	}

	return results
}
