package status

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/couchbaselabs/cbmultimanager/couchbase"
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/slice"
)

// bucketEndpointCheckers groups all the checkers that work on the data from the /pools/default/buckets endpoint as to
// avoid doing multiple REST calls.
func bucketEndpointCheckers(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	client, err := couchbase.NewClient(cluster.NodesSummary.GetHosts(), cluster.User, cluster.Password,
		cluster.GetTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("could not create client to communicate with clsuter: %w", err)
	}

	var dataNodes int
	for _, node := range cluster.NodesSummary {
		if slice.ContainsString(node.Services, "kv") {
			dataNodes++
		}
	}

	callTime := time.Now()
	buckets, err := client.GetPoolsBucket()
	if err != nil {
		return nil, fmt.Errorf("could not get pools/default/buckets: %w", err)
	}

	results := missingActiveVBuckets(buckets, callTime, cluster.UUID)
	results = append(results, missingVBucketReplicas(buckets, callTime, cluster.UUID)...)
	results = append(results, maxBuckets(buckets, callTime, cluster.UUID))
	results = append(results, replicavBucketNumber(buckets, callTime, cluster.UUID, dataNodes)...)
	return results, nil
}

func bucketStatCheckers(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	client, err := couchbase.NewClient(cluster.NodesSummary.GetHosts(), cluster.User, cluster.Password,
		cluster.GetTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("could not create client to communicate with cluster: %w", err)
	}

	results := make([]*values.WrappedCheckerResult, 0)
	for _, bucket := range cluster.BucketsSummary {
		callTime := time.Now()
		bucketStats, err := client.GetBucketStats(bucket.Name)
		if err != nil {
			results = append(results, &values.WrappedCheckerResult{Cluster: cluster.UUID, Bucket: bucket.Name, Error: err})
			continue
		}

		results = append(results, residentRatioTooLowCheck(bucketStats, bucket.Name, callTime, cluster.UUID))
		results = append(results, bucketMemoryUsageCheck(bucketStats, bucket.Name, callTime, cluster.UUID,
			bucket.Quota))
	}

	return results, nil
}

func maxBuckets(buckets []couchbase.Bucket, callTime time.Time, clusterUUID string) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "maxBuckets",
			Time:   callTime,
			Status: values.GoodCheckerStatus,
		},
		Cluster: clusterUUID,
	}

	result.Result.Value = []byte(fmt.Sprintf(`{"num_buckets":%d}`, len(buckets)))

	if len(buckets) > 30 {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = "Reduce the number of buckets as current number could cause performance " +
			"degradation."
	}

	return result
}

func missingActiveVBuckets(buckets []couchbase.Bucket, callTime time.Time,
	clusterUUID string) []*values.WrappedCheckerResult {
	results := make([]*values.WrappedCheckerResult, 0, len(buckets))

	for _, bucket := range buckets {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "missingActiveVBuckets",
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

func missingVBucketReplicas(buckets []couchbase.Bucket, callTime time.Time,
	clusterUUID string) []*values.WrappedCheckerResult {
	results := make([]*values.WrappedCheckerResult, 0, len(buckets))

	for _, bucket := range buckets {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "missingReplicaVBuckets",
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
	clusterUUID string) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "residentRatioTooLow",
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

func replicavBucketNumber(buckets []couchbase.Bucket, callTime time.Time, clusterUUID string,
	nodeCount int) []*values.WrappedCheckerResult {
	results := make([]*values.WrappedCheckerResult, 0, len(buckets))
	for _, bucket := range buckets {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "replicavBucketNumber",
				Status: values.GoodCheckerStatus,
				Time:   callTime,
			},
			Cluster: clusterUUID,
			Bucket:  bucket.Name,
		}

		if bucket.VBucketServerMap.NumReplicas >= 3 && nodeCount < 10 {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = fmt.Sprintf("Data nodes required: %d is not suitable for number of replicas:"+
				" %d. 10 or more nodes recommended for 3 replicas.", bucket.VBucketServerMap.NumReplicas, nodeCount)
		} else if bucket.VBucketServerMap.NumReplicas >= 2 && nodeCount < 5 {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = fmt.Sprintf("Data nodes required: %d is not suitable for number of replicas:"+
				" %d. 5 or more nodes recommended for 2 replicas.", bucket.VBucketServerMap.NumReplicas, nodeCount)
		}

		results = append(results, result)
	}

	return results
}

func bucketMemoryUsageCheck(bucketStat *values.BucketStat, bucket string, callTime time.Time,
	clusterUUID string, quota uint64) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "bucketMemoryUsage",
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
