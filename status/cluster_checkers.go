package status

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/couchbaselabs/cbmultimanager/couchbase"
	"github.com/couchbaselabs/cbmultimanager/values"
)

func singleOrTwoNodeClusterCheck(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "singleOrTwoNodeCluster",
			Time:   cluster.LastUpdate,
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
	}

	if len(cluster.NodesSummary) < 3 {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = "Production clusters should have at least three nodes. See " +
			"https://docs.couchbase.com/server/current/install/deployment-considerations-lt-3nodes.html for " +
			"more information."
	}

	return []*values.WrappedCheckerResult{result}, nil
}

func mixedModeCheck(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "mixedMode",
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
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = "The cluster is in mixed version mode. Please upgrade the nodes."
	}

	return []*values.WrappedCheckerResult{result}, nil
}

// sharedClientChecks groups a lot of checks that as to avoid creating multiple clients as the process of creating the
// client can result in doing multiple REST calls
func sharedClientChecks(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	client, err := couchbase.NewClient(cluster.NodesSummary.GetHosts(), cluster.User, cluster.Password,
		cluster.GetTLSConfig())
	if err != nil {
		return nil, fmt.Errorf("could not create client to communicate with clsuter: %w", err)
	}

	return []*values.WrappedCheckerResult{
		serverQuotaCheck(client),
		globalAutoCompactionCheck(client),
		autoFailoverChecker(client),
		dataLossChecker(client),
		backupLocationCheck(client),
	}, nil
}

func serverQuotaCheck(client couchbase.ClientIFace) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "serverQuota",
			Time:   client.GetBootstrap(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: client.GetClusterInfo().ClusterUUID,
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
	if err := json.Unmarshal(client.GetClusterInfo().PoolsRaw, &overlay); err != nil {
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

func globalAutoCompactionCheck(client couchbase.ClientIFace) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "globalAutoCompaction",
			Time:   client.GetBootstrap(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: client.GetClusterInfo().ClusterUUID,
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

	if err := json.Unmarshal(client.GetClusterInfo().PoolsRaw, &autoCompactionSettingsOverlay); err != nil {
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
		result.Result.Status = values.AlertCheckerStatus
		result.Result.Remediation = "Enable auto-compaction by providing a threshold in the settings page."
	}

	return result
}

func autoFailoverChecker(client couchbase.ClientIFace) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "autoFailoverEnabled",
			Time:   time.Now(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: client.GetClusterInfo().ClusterUUID,
	}

	autoFailoverSettings, err := client.GetAutoFailOverSettings()
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

func dataLossChecker(client couchbase.ClientIFace) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "dataLoss",
			Time:   time.Now(),
			Status: values.GoodCheckerStatus,
		},
		Cluster: client.GetClusterInfo().ClusterUUID,
	}

	logEntries, err := client.GetUILogs()
	if err != nil {
		result.Error = err
		return result
	}

	// traverse the entries backwards and try to find a match
	for i := len(logEntries) - 1; i >= 0; i-- {
		if strings.Contains(strings.ToLower(logEntries[i].Text), "lost data") {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Value, _ = json.Marshal(&logEntries[i])
			result.Result.Remediation = "Data loss messages where observed please cluster status."
			break
		}
	}

	return result
}

func activeClusterCheck(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "activeCluster",
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

func asymmetricalClusterCheck(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "asymmetricalCluster",
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
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = strings.Join(remediation, "\n")
	}

	return []*values.WrappedCheckerResult{result}, nil
}

func backupLocationCheck(client couchbase.ClientIFace) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "backupLocationCheck",
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
			result.Result.Status = values.AlertCheckerStatus
			result.Result.Remediation = "Number of backup location errors has increased." +
				"Ensure the backup service has constant access to the archive locations."
			break
		}
	}

	return result
}
