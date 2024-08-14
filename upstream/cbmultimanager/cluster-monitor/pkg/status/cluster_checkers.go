// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package status

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/tools-common/cbvalue"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func singleOrTwoNodeClusterCheck(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckSingleOrTwoNodeCluster,
			Time:   cluster.LastUpdate,
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	if len(cluster.NodesSummary) < 3 {
		result.Result.Status = values.InfoCheckerStatus
		result.Result.Remediation = "Production clusters should have at least three nodes. See " +
			"https://docs.couchbase.com/server/current/install/deployment-considerations-lt-3nodes.html for " +
			"more information."
	}

	return []*values.WrappedCheckerResult{result}, nil
}

func mixedModeCheck(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckMixedMode,
			Time:   cluster.LastUpdate,
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	versions := make(map[string][]string)
	for _, node := range cluster.NodesSummary {
		_, ok := versions[node.Version]
		if !ok {
			versions[node.Version] = []string{node.NodeUUID}
		} else {
			versions[node.Version] = append(versions[node.Version], node.NodeUUID)
		}
	}

	out, err := json.Marshal(versions)
	if err != nil {
		return nil, fmt.Errorf("could not marshal checker results: %w", err)
	}

	result.Result.Value = out
	if len(versions) > 1 {
		result.Result.Status = values.InfoCheckerStatus
		result.Result.Remediation = "The cluster is in mixed version mode. Please upgrade the nodes."
	}

	return []*values.WrappedCheckerResult{result}, nil
}

func cacheRESTDataChecks(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	checkers := map[string]func(cluster values.CouchbaseCluster) *values.WrappedCheckerResult{
		values.CheckServerQuota:               serverQuotaCheck,
		values.CheckGlobalAutoCompaction:      globalAutoCompactionCheck,
		values.CheckAutoFailoverEnabled:       autoFailoverChecker,
		values.CheckDataLoss:                  dataLossChecker,
		values.CheckEmptyServerGroup:          emptyGroupCheck,
		values.CheckDeveloperPreview:          developerPreviewCheck,
		values.CheckImbalancedIndexPartitions: imbalancedIndexPartitionsCheck,
		values.CheckTooManySearchReplicas:     checkTooManySearchReplicas,
		values.BucketCountChecks:              bucketCountChecks,
	}

	allRes := make([]*values.WrappedCheckerResult, 0, len(checkers))
	for checkerName, checkerFn := range checkers {
		if res := versionCheck(checkerName, &cluster, values.AllCheckerDefs); res != nil {
			allRes = append(allRes, res)
			continue
		}

		allRes = append(allRes, checkerFn(cluster))
	}

	indexResults, err := indexesChecks(cluster)
	if err != nil {
		return nil, err
	}
	allRes = append(allRes, indexResults...)

	return allRes, nil
}

// sharedClientChecks groups a lot of checks that as to avoid creating multiple clients as the process of creating the
// client can result in doing multiple REST calls
func sharedClientChecks(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	client, err := couchbase.NewClient(cluster.NodesSummary.GetHosts(), cluster.User, cluster.Password,
		cluster.GetTLSConfig(), false)
	if err != nil {
		return nil, fmt.Errorf("could not create client to communicate with cluster: %w", err)
	}

	checkers := map[string]func(client couchbase.ClientIFace) *values.WrappedCheckerResult{
		values.CheckBackupLocation:     backupLocationCheck,
		values.CheckBackupTaskOrphaned: backupTaskOrphaned,
	}

	allRes := make([]*values.WrappedCheckerResult, 0, len(checkers))
	for checkerName, checkerFn := range checkers {
		if res := versionCheck(checkerName, &cluster, values.AllCheckerDefs); res != nil {
			allRes = append(allRes, res)
			continue
		}

		allRes = append(allRes, checkerFn(client))
	}

	return allRes, nil
}

func serverQuotaCheck(cluster values.CouchbaseCluster) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckServerQuota,
			Time:   time.Now(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	type poolsStorageFilter struct {
		StorageTotals struct {
			RAM struct {
				QuotaTotal uint64 `json:"quotaTotal"`
				Total      uint64 `json:"total"`
			} `json:"ram"`
		} `json:"storageTotals"`
	}

	var overlay poolsStorageFilter
	if err := json.Unmarshal(cluster.PoolsRaw, &overlay); err != nil {
		result.Error = err
		return result
	}

	serverQuota := float64(overlay.StorageTotals.RAM.QuotaTotal) / float64(overlay.StorageTotals.RAM.Total) * 100
	result.Result.Value = []byte(fmt.Sprintf("{\"quota\":%.2f}", serverQuota))
	remediation := "Server Quota is approaching or exceeding 80% of available RAM. This may prevent OS function" +
		". Please reduce Server Quota."

	if serverQuota >= 85 {
		result.Result.Status = values.AlertCheckerStatus
		result.Result.Remediation = remediation
	} else if serverQuota >= 80 {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = remediation
	}

	return result
}

func emptyGroupCheck(cluster values.CouchbaseCluster) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckEmptyServerGroup,
			Time:   time.Now(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	serverGroups, err := cluster.GetCacheServerGroups()
	if err != nil {
		result.Error = err
		return result
	}

	emptyGroups := make([]string, 0)
	for _, groups := range serverGroups {
		if len(groups.Nodes) == 0 {
			emptyGroups = append(emptyGroups, groups.Name)
		}
	}

	if len(emptyGroups) > 0 {
		result.Result.Status = values.InfoCheckerStatus
		result.Result.Value, _ = json.Marshal(emptyGroups)
		result.Result.Remediation = "Please remove any empty server groups."
	}

	return result
}

func developerPreviewCheck(cluster values.CouchbaseCluster) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckDeveloperPreview,
			Time:   time.Now(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	if cluster.DeveloperPreview {
		result.Result.Status = values.AlertCheckerStatus
		result.Result.Remediation = "If this is a development only cluster, this alert may be disregarded. " +
			"Developer Preview mode is unsupported and should not be used in production, " +
			"create a new Couchbase Server cluster which does not have Developer Preview mode enabled"
	}
	return result
}

func globalAutoCompactionCheck(cluster values.CouchbaseCluster) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckGlobalAutoCompaction,
			Time:   time.Now(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	autoCompactionSettingsOverlay := struct {
		AutoCompactionSettings struct {
			DatabaseFragmentation struct {
				// sometimes this is the string "undefined" which makes unmarshalling it fun
				Percentage json.RawMessage `json:"percentage"`
				Size       json.RawMessage `json:"size"`
			} `json:"databaseFragmentationThreshold"`
		} `json:"autoCompactionSettings"`
	}{}

	if err := json.Unmarshal(cluster.PoolsRaw, &autoCompactionSettingsOverlay); err != nil {
		result.Error = fmt.Errorf("could not unmarshal autocompaction settings from pools/default: %w", err)
		return result
	}

	var valueSet int
	compactionSet := json.Unmarshal(
		autoCompactionSettingsOverlay.AutoCompactionSettings.DatabaseFragmentation.Percentage,
		&valueSet) == nil
	compactionSet = json.Unmarshal(autoCompactionSettingsOverlay.AutoCompactionSettings.DatabaseFragmentation.Size,
		&valueSet) == nil || compactionSet

	if !compactionSet {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = "Enable auto-compaction by providing a threshold in the settings page."
	}

	return result
}

func autoFailoverChecker(cluster values.CouchbaseCluster) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckAutoFailoverEnabled,
			Time:   time.Now(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	autoFailoverSettings, err := cluster.GetCacheAutoFailOverSettings()
	if err != nil {
		result.Error = err
		return result
	}

	result.Result.Value, _ = json.Marshal(autoFailoverSettings)

	if !autoFailoverSettings.Enabled {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = "Autofailover is disabled. Set a timeout to re-enable autofailover."
	}

	return result
}

func dataLossChecker(cluster values.CouchbaseCluster) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckDataLoss,
			Time:   time.Now(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	logEntries, err := cluster.GetCacheUILogs()
	if err != nil {
		result.Error = err
		return result
	}

	// traverse the entries backwards and try to find a match
	for i := len(logEntries) - 1; i >= 0; i-- {
		if strings.Contains(strings.ToLower(logEntries[i].Text), "lost data") {
			result.Result.Status = values.AlertCheckerStatus
			result.Result.Value, _ = json.Marshal(&logEntries[i])
			result.Result.Remediation = "Data loss messages were observed, please check cluster status."
			break
		}
	}

	return result
}

func activeClusterCheck(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckActiveCluster,
			Time:   cluster.LastUpdate,
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	unhealthy := make([]string, 0, len(cluster.NodesSummary))
	for _, node := range cluster.NodesSummary {
		if node.ClusterMembership != "active" || node.Status != "healthy" {
			unhealthy = append(unhealthy, node.Host)
		}
	}

	if len(unhealthy) != 0 {
		result.Result.Status = values.AlertCheckerStatus
		result.Result.Remediation = "Some nodes in the cluster are inactive, " +
			"please fix nodes or rebalance them out of cluster."

		type Nodes struct {
			Nodes []string `json:"nodes"`
		}

		nodes := &Nodes{Nodes: unhealthy}
		out, err := json.Marshal(nodes)
		if err != nil {
			result.Error = fmt.Errorf("could not marshal node information: %w", err)
		}

		result.Result.Value = out
	}

	return []*values.WrappedCheckerResult{result}, nil
}

func asymmetricalClusterCheck(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckAsymmetricalCluster,
			Time:   cluster.LastUpdate,
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	serviceNodes := make(map[string][]int)
	for i, node := range cluster.NodesSummary {
		for _, service := range node.Services {
			serviceNodes[service] = append(serviceNodes[service], i)
		}
	}

	var remediation []string
	for service, nodeList := range serviceNodes {
		if len(nodeList) <= 1 {
			continue
		}

		memTotal := cluster.NodesSummary[nodeList[0]].MemTotal
		swapTotal := cluster.NodesSummary[nodeList[0]].SwapTotal

		for i := 1; i < len(nodeList); i++ {
			if memTotal != cluster.NodesSummary[nodeList[i]].MemTotal ||
				swapTotal != cluster.NodesSummary[nodeList[i]].SwapTotal {
				remediation = append(remediation, fmt.Sprintf("Not all %s nodes have the same hardware,"+
					"make sure all of them have the same Memory and CPU quotas.", service))
				break
			}
		}
	}

	if len(remediation) > 0 {
		result.Result.Status = values.InfoCheckerStatus
		result.Result.Remediation = strings.Join(remediation, "\n")
	}

	return []*values.WrappedCheckerResult{result}, nil
}

func backupLocationCheck(client couchbase.ClientIFace) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckBackupLocation,
			Time:   client.GetBootstrap(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: client.GetClusterInfo().ClusterUUID,
	}

	t := time.Now()
	location, err := client.GetMetric(t.AddDate(0, 0, -3).Format(time.RFC3339), t.Format(time.RFC3339),
		"backup_location_check", "10m")
	if err != nil {
		result.Error = err
		return result
	}

	if len(location.Values) == 0 {
		return result
	}

	start, err := strconv.ParseInt(location.Values[0].Value, 10, 64)
	if err != nil {
		result.Error = fmt.Errorf("could not parse value: %w", err)
		return result
	}

	for _, item := range location.Values {
		val, err := strconv.ParseInt(item.Value, 10, 64)
		if err != nil {
			result.Error = fmt.Errorf("could not parse value: %w", err)
			break
		}

		if val > start {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = "Number of backup location errors has increased." +
				"Ensure the backup service has constant access to the archive locations."
			break
		}
	}

	return result
}

func backupTaskOrphaned(client couchbase.ClientIFace) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckBackupTaskOrphaned,
			Time:   client.GetBootstrap(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: client.GetClusterInfo().ClusterUUID,
	}

	t := time.Now()
	orphaned, err := client.GetMetric(t.AddDate(0, 0, -3).Format(time.RFC3339), t.Format(time.RFC3339),
		"backup_task_orphaned", "10m")
	if err != nil {
		result.Error = err
		return result
	}

	if len(orphaned.Values) == 0 {
		return result
	}

	start, err := strconv.ParseInt(orphaned.Values[0].Value, 10, 64)
	if err != nil {
		result.Error = fmt.Errorf("could not parse value: %w", err)
		return result
	}

	for _, item := range orphaned.Values {
		val, err := strconv.ParseInt(item.Value, 10, 64)
		if err != nil {
			result.Error = fmt.Errorf("could not parse value: %w", err)
			break
		}

		if val > start {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = fmt.Sprintf("Number of orphaned backup tasks for repository: %s"+
				" has increased. Check backup service tasks for more information.", orphaned.Repo)
			break
		}
	}

	return result
}

func indexesChecks(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	indexes, err := cluster.GetCacheIndexStatus()
	if err != nil {
		return nil, err
	}

	parsedIndexes, err := parseIndexes(indexes)
	if err != nil {
		return nil, err
	}

	var numIndexNodes int
	for _, node := range cluster.NodesSummary {
		if node.HasService("index") {
			numIndexNodes++
		}
	}

	// Now go over each and apply our checks.
	// We don't need to make() these slice beforehand, as append(nil, ...) will make it for us.
	var noRedundant, badRedundancy, tooManyReplicas []*values.IndexStatus
	for _, equiv := range parsedIndexes.equivalentIndexes {
		if len(equiv) == 1 {
			noRedundant = append(noRedundant, equiv[0])
		}

		// Find the number of equivalent indexes on each node
		// (we're working in the context of a single "equivalent group",
		// so this will be repeated for each set of equivalent indexes).
		nodes := make(map[string]uint)
		for _, idx := range equiv {
			// TODO: this doesn't handle partitioned indexes - I'm not sure on the best way of handling them
			if len(idx.Hosts) != 1 {
				continue
			}
			nodes[idx.Hosts[0]]++
			// While we're here (to avoid looping over all the indexes yet again), check the replica count. Alert
			// tooManyReplicas if we have more replicas than there are index nodes. First, though, check whether this is
			// the primary or the replica, otherwise we'll alert more than once for the same "base" index.
			// This string match is safe - parens aren't permitted in user-defined index names
			if !strings.Contains(idx.Name, "(replica") {
				if idx.NumReplica+1 > numIndexNodes {
					tooManyReplicas = append(tooManyReplicas, idx)
				}
			}
		}
		// Alert badRedundancy if any equivalent indexes on the same node are found.
		for node, count := range nodes {
			if count > 1 {
				badIndexes := make([]*values.IndexStatus, count)
				var i int
				for _, idx := range equiv {
					if idx.Hosts[0] == node {
						badIndexes[i] = idx
						i++
					}
				}
				badRedundancy = append(badRedundancy, badIndexes...)
			}
		}
	}

	return []*values.WrappedCheckerResult{
		makeIndexCheckerResult(values.CheckIndexWithNoRedundancy, noRedundant, cluster, values.WarnCheckerStatus,
			"Indexes without replicas or equivalent indexes can cause query failures if an index node is"+
				" failed over. Add replicas or equivalent indexes."),
		makeIndexCheckerResult(values.CheckBadRedundantIndex, badRedundancy, cluster, values.WarnCheckerStatus,
			"Indexes with replicas or equivalent indexes on the same node don't provide any redundancy."+
				" Move them to different Index Service nodes."),
		makeIndexCheckerResult(values.CheckTooManyIndexReplicas, tooManyReplicas, cluster, values.WarnCheckerStatus,
			"Indexes have more replicas defined than Index Service nodes present. Either add more nodes or"+
				" remove some replicas."),
		makeIndexCheckerResult(values.CheckMissingIndexPartitions, parsedIndexes.indexesWithMissingPartitions, cluster,
			values.AlertCheckerStatus, "There are fewer index partitions than were originally defined when making the index."+
				" Check if a node has been failed over. If this is not the case, recreate the index again"+
				" and contact Couchbase Technical Support."),
	}, nil
}

const imbalancedWarnPercent = 0.2

func imbalancedIndexPartitionsCheck(cluster values.CouchbaseCluster) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckImbalancedIndexPartitions,
			Status: values.GoodCheckerStatus,
			Time:   time.Now(),
		},
		Cluster: cluster.UUID,
	}

	indexesStorageStats, err := cluster.GetCacheIndexStorageStats()
	if err != nil {
		result.Error = fmt.Errorf("could not get index storage stats data: %w", err)
		return result
	}

	// indexMemory maps index names to a map of index partition numbers to the
	// amount of memory that each index partition takes up.
	indexMemory := make(map[string]map[int]int)
	// imbalancedPartitions stores all indexes in `indexMemory` which are
	// imbalanced as such:
	// index name -> partitionNumber -> partition memory
	imbalancedPartitions := make(map[string]map[int]int)
	for _, indexPartition := range indexesStorageStats {
		if _, ok := indexMemory[indexPartition.Name]; !ok {
			indexMemory[indexPartition.Name] = make(map[int]int)
		}
		indexMemory[indexPartition.Name][indexPartition.PartitionID] = indexPartition.Stats.IndexMemory
	}
	for name, partitionMap := range indexMemory {
		max := math.Inf(-1)
		min := math.Inf(1)
		for _, memory := range partitionMap {
			memoryFloat := float64(memory)
			if memoryFloat > max {
				max = memoryFloat
			}
			if memoryFloat < min {
				min = memoryFloat
			}
		}
		if ((max - min) / max) > imbalancedWarnPercent {
			imbalancedPartitions[name] = partitionMap
		}
	}

	if len(imbalancedPartitions) > 0 {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = "Recreate imbalanced index to redistribute index partition data," +
			" making sure the index partitions are hashed to valid fields."
		result.Result.Value, _ = json.Marshal(imbalancedPartitions)
	}

	return result
}

func checkDuplicateNodeUUIDs(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckDuplicateNodeUUID,
			Time:   cluster.LastUpdate,
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	uids := make(map[string]int)
	dup := make([]string, 0)
	for _, node := range cluster.NodesSummary {
		uids[node.NodeUUID]++

		if uids[node.NodeUUID] == 2 {
			dup = append(dup, node.NodeUUID)
		}
	}

	if len(dup) > 0 {
		result.Result.Status = values.AlertCheckerStatus
		result.Result.Remediation = "Contact Couchbase Technical Support."
		result.Result.Value, _ = json.Marshal(dup)
	}

	return []*values.WrappedCheckerResult{result}, nil
}

func checkTooManySearchReplicas(cluster values.CouchbaseCluster) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckTooManySearchReplicas,
			Time:   time.Now(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	ftsNodes := 0
	for _, node := range cluster.NodesSummary {
		if node.HasService("fts") {
			ftsNodes++
		}
	}

	if ftsNodes == 0 {
		return result
	}

	indexes, err := cluster.GetCacheFTSIndexStatus()
	if err != nil {
		result.Error = err
		return result
	}

	misconfigured := make([]string, 0)
	for _, index := range indexes.IndexDefs.IndexDefs {
		if index.PlanParameters.NumReplicas > (ftsNodes - 1) {
			misconfigured = append(misconfigured, index.Name)
		}
	}

	if len(misconfigured) != 0 {
		sort.Strings(misconfigured)
		result.Result.Status = values.AlertCheckerStatus
		result.Result.Remediation = "Ensure there are fewer FTS index replicas than " +
			"nodes running the Search Service."
		result.Result.Value, _ = json.Marshal(misconfigured)
	}

	return result
}

const (
	coresPerBucketLimitv7          = 0.2
	coresPerBucketLimitv6andBelow  = 0.4
	maxAllowedBucketCountv6_5      = 30
	maxAllowedBucketCountBelowv6_5 = 10
)

func bucketCountChecks(cluster values.CouchbaseCluster) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.BucketCountChecks,
			Time:   time.Now(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	bucketSummary := cluster.BucketsSummary

	numberOfBuckets := len(bucketSummary)
	minVersion := cluster.NodesSummary.GetMinVersion()
	minCPUCount := math.MaxInt16
	var nodeWithLeastCPUs string

	for _, node := range cluster.NodesSummary {
		if node.CPUCount < minCPUCount && node.HasService("kv") {
			minCPUCount = node.CPUCount
			nodeWithLeastCPUs = node.Host
		}
	}

	// To avoid any division errors.
	if numberOfBuckets == 0 {
		return result
	}

	coresPerBucket := float32(minCPUCount) / float32(numberOfBuckets)

	switch {
	// First two cases are for number of bucket count not exceeding recommended.
	case numberOfBuckets > maxAllowedBucketCountBelowv6_5 && minVersion.AtLeast(cbvalue.Version6_5_0):
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = fmt.Sprintf("The number of Buckets is %v. "+
			"Maximum allowed is 30. Please reduce the bucket count.", numberOfBuckets)
	case numberOfBuckets > maxAllowedBucketCountv6_5 && minVersion.Older(cbvalue.Version6_5_0):
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = fmt.Sprintf("The number of Buckets is %v. "+
			"Maximum allowed is 10. Please reduce the bucket count.", numberOfBuckets)
	// Next two cases are for enforcing the cores/bucket requirement.
	case coresPerBucket <= coresPerBucketLimitv6andBelow && minVersion.Older(cbvalue.Version7_0_0):
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = fmt.Sprintf("The number of CPUs per bucket for node %v is %.2g. "+
			"Please increase available processors or reduce the bucket count.", nodeWithLeastCPUs, coresPerBucket)
	case coresPerBucket <= coresPerBucketLimitv7 && minVersion.AtLeast(cbvalue.Version7_0_0):
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = fmt.Sprintf("The number of CPUs per bucket for node %v is %.2g. "+
			"Please increase available processors or reduce the bucket count.", nodeWithLeastCPUs, coresPerBucket)
	// This case is for one core per bucket INFO.
	case minCPUCount < numberOfBuckets:
		result.Result.Status = values.InfoCheckerStatus
		result.Result.Remediation = fmt.Sprintf("Number of CPUs is less than number of buckets for node %v,"+
			" please increase available processors or reduce bucket count.", nodeWithLeastCPUs)
	}

	return result
}
