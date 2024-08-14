// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package status

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/cbvalue"
	"github.com/couchbase/tools-common/netutil"
	"go.uber.org/zap"
)

func checkAnalyticsJREVersion(jreVersion string, jreVendor string, cbVersion string) (*values.CheckerResult, error) {
	createJREVersionResult := func(value json.RawMessage, status values.CheckerStatus,
		remediation string,
	) *values.CheckerResult {
		return &values.CheckerResult{
			Name:        values.CheckAnalyticsJRE,
			Value:       value,
			Status:      status,
			Remediation: remediation,
		}
	}
	versionValue, err := json.Marshal(jreVersion)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshall JRE version: %w", err)
	}
	if jreVersion == "" {
		return createJREVersionResult(versionValue,
			values.WarnCheckerStatus, "Could not determine JRE version, string is empty"), nil
	}
	delim := "."
	if strings.Contains(jreVersion, "_") {
		delim = "_"
	}

	if len(strings.Split(jreVersion, delim)) < 2 {
		return createJREVersionResult(versionValue,
			values.WarnCheckerStatus, "Could not determine JRE version, failed to parse major and minor version"), nil
	}

	majorVersion := strings.Split(jreVersion, delim)[0]
	minorVersion, err := strconv.Atoi(strings.Split(jreVersion, delim)[1])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse minor version: %v", err)
	}
	if cbvalue.Version(cbVersion).Older(cbvalue.Version6_6_0) {
		vendor := values.FindSupportedVendor(values.SupportedVendorsCB6_5AndBelow, jreVendor)
		if vendor == nil {
			return createJREVersionResult(versionValue, values.InfoCheckerStatus,
				"Could not determine JRE version support, invalid JRE vendor"), nil
		}
		for _, version := range vendor.SupportedVersion {
			if version.Major == majorVersion {
				if version.Minor > minorVersion {
					return createJREVersionResult(versionValue, values.AlertCheckerStatus,
						fmt.Sprintf("JRE update should be %d for Java %s", version.Minor, version.Major)), nil
				}
				return createJREVersionResult(versionValue, values.GoodCheckerStatus, ""), nil
			}
		}
		return createJREVersionResult(versionValue, values.AlertCheckerStatus,
			"JRE Version should be JRE 8 or 11"), nil
	}

	if vendor := values.FindSupportedVendor(values.SupportedVendorsCB6_6AndAbove, jreVendor); vendor != nil {
		for _, version := range vendor.SupportedVersion {
			if majorVersion != version.Major {
				return createJREVersionResult(versionValue,
					values.AlertCheckerStatus, "JRE version should be JRE 11"), nil
			}
		}

		return createJREVersionResult(versionValue, values.GoodCheckerStatus, ""), nil
	}

	return createJREVersionResult(versionValue, values.InfoCheckerStatus, ""), nil
}

// helper function for checkAnalyticsJRE
// checks just that the vendor is valid
func checkAnalyticsJREVendor(jreVendor string) (*values.CheckerResult, error) {
	createJREVendorResult := func(value json.RawMessage, status values.CheckerStatus,
		remediation string,
	) *values.CheckerResult {
		return &values.CheckerResult{
			Name:        values.CheckAnalyticsJRE,
			Value:       value,
			Status:      status,
			Remediation: remediation,
		}
	}

	vendorValue, err := json.Marshal(jreVendor)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshall java vendor: %w", err)
	}

	switch jreVendor {
	case "":
		return createJREVendorResult(vendorValue, values.WarnCheckerStatus,
			"Could not determine JRE vendor"), nil

	case "Oracle Corporation", "AdoptOpenJDK", "Eclipse Foundation":
		return createJREVendorResult(vendorValue, values.GoodCheckerStatus, ""), nil

	default:
		return createJREVendorResult(vendorValue, values.AlertCheckerStatus,
			fmt.Sprintf("%s (Unsupported)", jreVendor)), nil
	}
}

// Checks Analytics Service is running a supported version of Java created by a supported vendor
func checkAnalyticsJRE(cluster *values.CouchbaseCluster, node *values.NodeSummary,
	diagnostics *values.AnalyticsNodeDiagnostics) (
	[]*values.WrappedCheckerResult, error,
) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))

	// helper function for appending the wrapped results
	appendToResults := func(result *values.CheckerResult) {
		result.Time = cluster.LastUpdate
		results = append(results, &values.WrappedCheckerResult{
			Result:  result,
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		})
	}

	jreVendor := diagnostics.Runtime.SystemProperties.JavaVendor
	jreVersion := diagnostics.Runtime.SystemProperties.JavaVersion

	vendorResult, err := checkAnalyticsJREVendor(jreVendor)
	if err != nil {
		return nil, err
	}
	appendToResults(vendorResult)
	versionResult, err := checkAnalyticsJREVersion(jreVersion, jreVendor, node.Version)
	if err != nil {
		return nil, err
	}
	appendToResults(versionResult)

	return results, nil
}

func oneServicePerNodeCheck(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	remediation := "Production environments should run single services on each node. See " +
		"https://docs.couchbase.com/server/current/learn/services-and-indexes/services/services.html for" +
		" more information."

	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckOneServicePerNode,
				Status: values.GoodCheckerStatus,
				Time:   cluster.LastUpdate,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		if len(node.Services) > 1 {
			result.Result.Status = values.InfoCheckerStatus
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

func unhealthyNodesCheck(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	remediation := "One or more nodes are marked as unhealthy or are not active members of the cluster. Please fix" +
		" the nodes or rebalance them in/out."

	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckUnhealthyNode,
				Status: values.GoodCheckerStatus,
				Time:   cluster.LastUpdate,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		if node.Status != "healthy" || node.ClusterMembership != "active" {
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

func supportedVersionCheck(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	maintenanceRemediation := "This node has reached End of Maintenance, please upgrade to maintain supportability."
	supportRemediation := "This node has reached End of Support, please upgrade to receive support."
	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckSupportedVersion,
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		currDate := time.Now()
		value, ok := values.GAVersions[strings.TrimSuffix(node.Version, "-enterprise")]
		if !ok {
			result.Result.Status = values.InfoCheckerStatus
			result.Result.Remediation = fmt.Sprintf("Unofficial version: '%s' is not supported. "+
				"Please upgrade to a supported version at http://couchbase.com/downloads", node.Version)
			results = append(results, result)
			continue
		}

		if currDate.After(value.EOS) {
			result.Result.Status = values.InfoCheckerStatus
			result.Result.Remediation = supportRemediation
			// Some versions' EOM date can be TBD
		} else if !value.EOM.IsZero() && currDate.After(value.EOM) {
			result.Result.Status = values.InfoCheckerStatus
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

func nonGABuildCheck(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	remediation := "This build is not generally available - please install a supported GA version."

	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckNonGABuild,
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		build := strings.TrimSuffix(node.Version, "-enterprise")
		if _, ok := values.GAVersions[build]; !ok {
			result.Result.Status = values.InfoCheckerStatus
			result.Result.Remediation = remediation
		}

		results = append(results, result)
	}

	return results, nil
}

func nodeSwapUsageCheck(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckNodeSwapUsage,
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		if node.SwapTotal > 0 {
			if float64(node.SwapUsed) >= float64(node.SwapTotal)*0.9 {
				result.Result.Status = values.AlertCheckerStatus
				result.Result.Remediation = "Swap usage is at or above 90% of available swap memory, " +
					"increase RAM or allocate additional memory."
			} else if node.SwapUsed != 0 {
				result.Result.Status = values.WarnCheckerStatus
				result.Result.Remediation = "Swap usage on this node is not 0 - please increase RAM or check memory usage."
			}
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

const minMemoryBytes = 4294967296

// belowMinMemCheck checks to see if a node has the minimum recommended RAM to run Couchbase.
func belowMinMemCheck(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckBelowMinMem,
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		if node.MemTotal < minMemoryBytes {
			result.Result.Status = values.InfoCheckerStatus
			result.Result.Remediation = "Less than the recommended 4GB of memory on node, " +
				"Please increase available memory to atleast 4GB"
		}
		results = append(results, result)
	}
	return results, nil
}

func runNodeSelfChecks(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	for _, node := range cluster.NodesSummary {
		storage, err := cluster.GetCacheNodeStorage()
		if err != nil {
			return nil, fmt.Errorf("could not get storage information: %w", err)
		}

		results = append(
			results,
			nodeDiskSpaceCheck(&cluster, storage, node),
			sharedFileSystemsCheck(&cluster, storage, node),
		)

		if node.HasService("index") {
			gsiSettings, err := cluster.GetCacheGSISettings()
			if err != nil {
				return nil, fmt.Errorf("could not get GSI Settings information: %w", err)
			}

			results = append(results, gsiSettingsChecks(&cluster, node, gsiSettings)...)
		}

		if node.HasService("cbas") {
			diagnostics, err := cluster.GetCacheAnalyticalNodeDiag()
			if err != nil {
				return nil, fmt.Errorf("could not get Analytics diagnostics: %w", err)
			}
			result, err := checkAnalyticsJRE(&cluster, &node, diagnostics)
			if err != nil {
				return nil, fmt.Errorf("could not check Analytics diagnostics: %w", err)
			}

			results = append(results, result...)
		}
	}

	return results, nil
}

func nodeDiskSpaceCheck(cluster *values.CouchbaseCluster, storage *values.Storage,
	node values.NodeSummary,
) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckNodeDiskSpace,
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

func splitPath(path string, isWindows bool) []string {
	separator := "/"
	if isWindows {
		separator = "\\"
		path = strings.ToLower(path)
	}
	// Linux prohibits / in a directory name, Windows prohibits both \ and /.
	parts := strings.Split(path, separator)
	// Handle the case where the path starts with a / (absolute paths)
	if parts[0] == "" && len(parts) > 1 {
		parts = parts[1:]
	}
	// And the case where there's a trailing slash
	if len(parts) > 1 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func isPathRootedInPartition(path, partition string, isWindows bool) bool {
	// For some reason CB turns backslashes into forward slashes for storage paths,
	// but not in availableStorage.
	var pathParts []string
	if isWindows {
		pathParts = splitPath(strings.ToLower(path), false)
	} else {
		pathParts = splitPath(path, false)
	}
	directoryParts := splitPath(partition, isWindows)
	if len(directoryParts) > len(pathParts) {
		// can't possibly be rooted
		return false
	}
	if len(directoryParts) == 1 && directoryParts[0] == "" {
		// that's just /
		return true
	}
	for i := 0; i < len(directoryParts); i++ {
		if pathParts[i] != directoryParts[i] {
			return false
		}
	}
	return true
}

func findDeepestRootFor(path string, roots []string, isWindows bool) string {
	longest := ""
	for _, root := range roots {
		if longest == "" || len(root) > len(longest) {
			if isPathRootedInPartition(path, root, isWindows) {
				longest = root
			}
		}
	}
	return longest
}

func sharedFileSystemsCheck(cluster *values.CouchbaseCluster, storage *values.Storage,
	node values.NodeSummary,
) *values.WrappedCheckerResult {
	isWindows := strings.HasPrefix(node.OS, "win")
	// Go through the services on the target node, find the deepest file system hosting its storage (they can be
	// nested), and mark it.
	allPartitionPaths := make([]string, 0)
	for _, store := range storage.Available.DiskStorage {
		allPartitionPaths = append(allPartitionPaths, store.Path)
	}
	servicesOnFileSystem := make(map[string][]string)
	allStores := append(storage.NodeStorage.HDD, storage.NodeStorage.SSD...)
	for _, store := range allStores {
		for svc, paths := range store.GetAllPaths() {
			if node.HasService(svc) {
				for _, servicePath := range paths {
					partition := findDeepestRootFor(servicePath, allPartitionPaths, isWindows)
					if _, ok := servicesOnFileSystem[partition]; !ok {
						servicesOnFileSystem[partition] = make([]string, 0)
					}
					servicesOnFileSystem[partition] = append(servicesOnFileSystem[partition], svc)
				}
			}
		}
	}

	// And check if any are duplicates
	issues := make([]string, 0)
	for path, svcs := range servicesOnFileSystem {
		// Special case: Indexing and FTS sharing a path is okay (they do that anyway)
		if ftsIdx := indexOf("fts", svcs); indexOf("index", svcs) > -1 && ftsIdx > -1 {
			// https://github.com/golang/go/wiki/SliceTricks#delete-without-preserving-order
			svcs[ftsIdx] = svcs[len(svcs)-1]
			svcs = svcs[:len(svcs)-1]
		}
		if len(svcs) > 1 {
			for i, svc := range svcs {
				svcs[i] = friendlyNameForService(svc)
			}
			// Need to sort it, because the order can be arbitrary
			sort.Slice(svcs, func(i, j int) bool {
				return svcs[i] < svcs[j]
			})
			issues = append(issues, fmt.Sprintf(
				"Services on %s: %s.",
				path,
				strings.Join(svcs, ", "),
			))
		}
	}

	result := &values.WrappedCheckerResult{
		Cluster: cluster.UUID,
		Node:    node.NodeUUID,
		Result: &values.CheckerResult{
			Name:   values.CheckSharedFilesystems,
			Time:   cluster.LastUpdate,
			Status: values.GoodCheckerStatus,
		},
	}

	if len(issues) > 0 {
		result.Result.Status = values.InfoCheckerStatus
		result.Result.Remediation = "Having multiple services on the same file system can cause I/O contention, " +
			"leading to reduced performance. For optimum throughput, place each service on its own file system."
		// Need to sort it, because the order can be arbitrary
		sort.Slice(issues, func(i, j int) bool {
			return issues[i] < issues[j]
		})
		value, err := json.Marshal(issues)
		if err != nil {
			result.Error = err
		} else {
			result.Result.Value = value
		}
	}
	return result
}

func gsiSettingsChecks(cluster *values.CouchbaseCluster,
	node values.NodeSummary, settings *values.GSISettings,
) []*values.WrappedCheckerResult {
	logLevelResult := values.WrappedCheckerResult{
		Cluster: cluster.UUID,
		Node:    node.NodeUUID,
		Result: &values.CheckerResult{
			Name:   values.CheckGSILogLevel,
			Time:   cluster.LastUpdate,
			Status: values.GoodCheckerStatus,
		},
	}
	var err error
	logLevelResult.Result.Value, err = settings.LogLevel.MarshalJSON()
	if err != nil {
		zap.S().Warnw("(GSI Settings Checks) Error when marshaling log level", "err", err)
		logLevelResult.Result.Status = values.MissingCheckerStatus
		return []*values.WrappedCheckerResult{&logLevelResult}
	}
	if settings.LogLevel != values.DefaultGSILogLevel {
		logLevelResult.Result.Status = values.WarnCheckerStatus
		logLevelResult.Result.Remediation = "Please change the log level to Info to ensure adequate " +
			"diagnostics are collected."
	}
	return []*values.WrappedCheckerResult{&logLevelResult}
}

// checkNodeServiceStatus confirms that all the services in each node are active.
func checkNodeServiceStatus(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	res := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))

	for _, node := range cluster.NodesSummary {
		client, err := couchbase.NewUnpopulatedClientForSingleNode([]string{node.Host}, cluster.User,
			cluster.Password, cluster.GetTLSConfig(), true)
		if err != nil {
			return nil, fmt.Errorf("could not create client: %w", err)
		}

		var failedServices []string
		for _, service := range node.Services {
			cbrestService, err := convertToCBRestService(service)
			if err != nil {
				zap.S().Errorw("(Status) Unknown service found, skipping", "service", service, "cluster", cluster.UUID,
					"node", node.NodeUUID)
				continue
			}

			// We only want to test the services that a client would test,
			// since we're likely running outside the cluster.
			switch cbrestService {
			case cbrest.ServiceManagement,
				cbrest.ServiceQuery,
				cbrest.ServiceSearch,
				cbrest.ServiceViews,
				cbrest.ServiceAnalytics:
				// Test it
			case cbrest.ServiceData:
				continue // TODO: KV needs special handling
			default:
				continue // Not a service a client would use
			}

			if err = client.PingService(cbrestService); err != nil {
				zap.S().Debugw("(Status) Error when pinging service",
					"service", cbrestService,
					"err", err)
				// Try and get the port which we failed to reach.
				port := " (unknown)"

				hosts, err := client.GetAllServiceHosts(cbrestService)
				if err == nil && len(hosts) > 0 {
					_, portStr, err := net.SplitHostPort(netutil.TrimSchema(hosts[0]))
					if err == nil {
						port = fmt.Sprintf(" (%s)", portStr)
					}
				}

				failedServices = append(failedServices, service+port)
			}
		}

		nodeRes := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name: values.CheckServiceStatus,
				Time: time.Now(),
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		if len(failedServices) == 0 {
			nodeRes.Result.Status = values.GoodCheckerStatus
		} else {
			nodeRes.Result.Status = values.WarnCheckerStatus
			nodeRes.Result.Remediation = fmt.Sprintf("Could not connect to services %v, please check that the "+
				"required ports are available and that they are not been blocked a firewall.", failedServices)
			nodeRes.Result.Value, _ = json.Marshal(failedServices)
		}

		res = append(res, nodeRes)
	}

	return res, nil
}

const usedMemoryWarnPercent float64 = 90

// freeMemCheck checks to see what percentage of memory is free
func freeMemCheck(cluster values.CouchbaseCluster) ([]*values.WrappedCheckerResult, error) {
	results := make([]*values.WrappedCheckerResult, 0, len(cluster.NodesSummary))
	for _, node := range cluster.NodesSummary {
		result := &values.WrappedCheckerResult{
			Result: &values.CheckerResult{
				Name:   values.CheckFreeMem,
				Time:   cluster.LastUpdate,
				Status: values.GoodCheckerStatus,
			},
			Cluster: cluster.UUID,
			Node:    node.NodeUUID,
		}

		memUsedPercentage := 100 * (1 - (float64(node.MemFree) / float64(node.MemTotal)))
		if memUsedPercentage > usedMemoryWarnPercent {
			result.Result.Status = values.WarnCheckerStatus
			result.Result.Remediation = "Add more RAM to the node, or review the resource usage " +
				"of other applications on the server."
		}
		result.Result.Value, _ = json.Marshal(fmt.Sprintf("%.2f%%", memUsedPercentage))
		results = append(results, result)
	}
	return results, nil
}

func convertToCBRestService(service string) (cbrest.Service, error) {
	switch service {
	case "kv":
		return cbrest.ServiceData, nil
	case "backup":
		return cbrest.ServiceBackup, nil
	case "cbas":
		return cbrest.ServiceAnalytics, nil
	case "index":
		return cbrest.ServiceGSI, nil
	case "fts":
		return cbrest.ServiceSearch, nil
	case "n1ql":
		return cbrest.ServiceQuery, nil
	case "eventing":
		return cbrest.ServiceEventing, nil
	case "views":
		return cbrest.ServiceViews, nil
	default:
		return "", fmt.Errorf("unknown service - %s", service)
	}
}

func friendlyNameForService(service string) string {
	switch service {
	case "kv":
		return "Data"
	case "index":
		return "Index"
	case "n1ql":
		return "Query"
	case "fts":
		return "Search"
	case "cbas":
		return "Analytics"
	case "eventing":
		return "Eventing"
	case "views":
		return "Views"
	case "backup":
		return "Backup"
	default:
		zap.S().Warnw("(Node Checkers) Missing friendly name", "service", service)
		return service
	}
}

func indexOf(target string, list []string) int {
	for i, str := range list {
		if str == target {
			return i
		}
	}
	return -1
}
