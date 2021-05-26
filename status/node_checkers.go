package status

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/couchbaselabs/cbmultimanager/couchbase"
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/slice"
)

func oneServicePerNodeCheck(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	remediation := "Production environments should run single services on each node. See " +
		"https://docs.couchbase.com/server/current/learn/services-and-indexes/services/services.html for" +
		" more information."

	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "oneServicePerNode",
				Status: values.GoodCheckerStatus,
				Time:   cluster.LastUpdate,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		if len(node.Services) > 1 {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = remediation
			out, err := json.Marshal(node)
			if err != nil {
				result.Error = fmt.Errorf("could not marshal node info: %w", err)
			}

			result.Result.Value = out
		}

		results = append(results, result)
	}

	return results, nil
}

func unhealthyNodesCheck(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	remediation := "One or more nodes are marked as unhealthy or are not active members of the cluster. Please fix" +
		" the nodes or rebalance them in/out."

	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "unhealthyNode",
				Status: values.GoodCheckerStatus,
				Time:   cluster.LastUpdate,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		if node.Status != "healthy" || node.ClusterMembership != "active" {
			result.Result.Status = values.AlertCheckerStatus
			result.Result.Remediation = remediation
			out, err := json.Marshal(node)
			if err != nil {
				result.Error = fmt.Errorf("could not marshal node info: %w", err)
			}

			result.Result.Value = out
		}

		results = append(results, result)
	}

	return results, nil
}

func supportedVersionCheck(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	maintenanceRemediation := "This node has reached End of Maintenance, please upgrade to maintain supportability."
	supportRemediation := "This node has reached End of Support, please upgrade to receive support."
	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "supportedVersion",
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		currDate := time.Now()
		value, ok := values.GAVersions[strings.TrimSuffix(node.Version, "-enterprise")]
		if !ok {
			result.Result.Status = values.AlertCheckerStatus
			result.Result.Remediation = fmt.Sprintf("Unofficial version: '%s' is not supported. "+
				"Please upgrade to a supported version at http://couchbase.com/downloads", node.Version)
			results = append(results, result)
			continue
		}

		if currDate.After(value.EOS) {
			result.Result.Status = values.AlertCheckerStatus
			result.Result.Remediation = supportRemediation
		} else if currDate.After(value.EOM) {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = maintenanceRemediation
		}

		out, err := json.Marshal(node.Version)
		if err != nil {
			result.Error = fmt.Errorf("could not marshal node info: %w", err)
		}

		result.Result.Value = out
		results = append(results, result)
	}

	return results, nil
}

func nonGABuildCheck(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	remediation := "This build is not generally available - please install a supported GA version."

	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "nonGABuild",
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		build := strings.TrimSuffix(node.Version, "-enterprise")
		if _, ok := values.GAVersions[build]; !ok {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = remediation
		}

		results = append(results, result)
	}

	return results, nil
}

func nodeSwapUsageCheck(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "nodeSwapUsage",
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		if float64(node.SwapUsed) >= float64(node.SwapTotal)*0.9 {
			result.Result.Status = values.AlertCheckerStatus
			result.Result.Remediation = "Swap usage is at or above 90% of available swap memory, " +
				"increase RAM or allocate additional memory."
		} else if node.SwapUsed != 0 {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = "Swap usage on this node is not 0 - please increase RAM or check memory usage."
		}

		out, err := json.Marshal(node.SwapUsed)
		if err != nil {
			result.Error = fmt.Errorf("could not marshal swap info: %w", err)
		}

		result.Result.Value = out
		results = append(results, result)
	}

	return results, nil
}

func cpuBucketCountCheck(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   "cpuBucketCount",
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		if !slice.ContainsString(node.Services, "kv") {
			continue
		}

		if node.CPUCount == 0 {
			result.Error = fmt.Errorf("could not retrieve cpu info")
		} else if node.CPUCount < len(cluster.BucketsSummary) {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = "Number of CPU's is less than number of buckets," +
				" please increase available processors or reduce bucket count."
		}

		results = append(results, result)
	}

	return results, nil
}

func runNodeSelfChecks(cluster *values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	for _, node := range cluster.NodesSummary {
		client, err := couchbase.NewClient([]string{node.Host}, cluster.User, cluster.Password,
			cluster.GetTLSConfig())
		if err != nil {
			return nil, fmt.Errorf("could not create client: %w", err)
		}

		storage, err := client.GetNodeStorage()
		if err != nil {
			return nil, fmt.Errorf("could not get storage information: %w", err)
		}

		results = append(results, nodeDiskSpaceCheck(cluster, storage, node))
	}

	return results, nil
}

func nodeDiskSpaceCheck(cluster *values.CouchbaseCluster, storage *values.Storage,
	node values.NodeSummary) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "nodeDiskSpace",
			Time:   cluster.LastUpdate,
			Status: values.GoodCheckerStatus,
		},
		Cluster: cluster.UUID,
		Node:    node.NodeUUID,
	}

	var bad []string
	for _, disk := range storage.Available.DiskStorage {
		if disk.Usage >= 90 {
			bad = append(bad, disk.Path)
		}
	}

	type disks struct {
		Disks []string `json:"disk"`
	}

	disk := &disks{Disks: bad}
	if len(bad) != 0 {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = fmt.Sprintf("Usage on disk(s): %s is at or above 90%%. Please assess "+
			"disk usage.", bad)
		out, err := json.Marshal(disk)
		if err != nil {
			result.Error = fmt.Errorf("could not marshal disk info: %w", err)
		}

		result.Result.Value = out
	}

	return result
}
