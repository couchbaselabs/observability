// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"time"

	"github.com/couchbase/tools-common/cbvalue"
	"github.com/couchbase/tools-common/netutil"

	// Blank embed for versionsfile
	_ "embed"
)

// HeartIssue is used to differentiate the reason a heartbeat failed.
type HeartIssue uint8

const (
	NoHeartIssue HeartIssue = iota
	BadAuthHeartIssue
	NoConnectionHeartIssue
	UUIDMismatchHeartIssue
)

type Status string

const (
	Waiting    Status = "waiting"
	Done       Status = "done"
	InProgress Status = "in progress"
)

type ClusterProgress struct {
	Status        Status     `json:"status"`
	Done          int        `json:"done"`
	Failed        int        `json:"failed"`
	TotalCheckers int        `json:"total_checkers"`
	Start         *time.Time `json:"start,omitempty"`
	End           *time.Time `json:"end,omitempty"`
}

type ClusterProgressMap map[string]*ClusterProgress

// ClusterInfo contains basic summary of details about the cluster hardware.
type ClusterInfo struct {
	RAMQuota       uint64 `json:"ram_quota"`
	RAMUsed        uint64 `json:"ram_used"`
	DiskTotal      uint64 `json:"disk_total"`
	DiskUsed       uint64 `json:"disk_used"`
	DiskUsedByData uint64 `json:"disk_used_by_data"`
}

// ClusterStatusSummary contains a summary of the checker results.
type ClusterStatusSummary struct {
	Good      uint64 `json:"good"`
	Warnings  uint64 `json:"warnings"`
	Alerts    uint64 `json:"alerts"`
	Info      uint64 `json:"info"`
	Missing   uint64 `json:"missing"`
	Dismissed uint64 `json:"dismissed"`
}

// XDCR remote cluster data struct
type RemoteClusters []RemoteCluster

type RemoteCluster struct {
	ConnectivityStatus string `json:"connectivity_status"`
	Hostname           string `json:"hostname"`
	Name               string `json:"name"`
	UUID               string `json:"uuid"`
	FromName           string `json:"from_name"`
	FromUUID           string `json:"from_uuid"`
}

type AutoFailoverSettings struct {
	Enabled bool `json:"enabled"`
}

type Bucket struct {
	Name             string           `json:"name"`
	VBucketServerMap VBucketServerMap `json:"vBucketServerMap"`
	StorageEngine    string           `json:"storageBackend"`
	MaxTTL           uint64           `json:"maxTTL,omitempty"`
}

type VBucketServerMap struct {
	ServerList  []string `json:"serverList"`
	NumReplicas int      `json:"numReplicas"`
	VBucketMap  [][]int  `json:"vBucketMap"`
}

type UILogs struct {
	List []UILogEntry `json:"list"`
}

type UILogEntry struct {
	Code       int    `json:"code"`
	Module     string `json:"module"`
	Node       string `json:"node"`
	ServerTime string `json:"serverTime"`
	Text       string `json:"text"`
	Type       string `json:"type"`
}

// Cache REST Data. This struct will contain all the data needed for checkers via REST API calls
type CacheRESTData struct {
	Buckets              []Bucket                  `json:"bucket"`
	BucketStats          []*BucketStat             `json:"bucket_stats"`
	ServerGroups         []ServerGroup             `json:"server_groups"`
	NodeStorage          *Storage                  `json:"node_storage"`
	AutoFailOverSettings *AutoFailoverSettings     `json:"auto_failover_settings"`
	IndexStatus          []*IndexStatus            `json:"index_status"`
	GSISettings          *GSISettings              `json:"gsi_settings"`
	IndexStorageStats    []*IndexStatsStorage      `json:"index_storage_stats"`
	FTSIndexStatus       FTSIndexStatus            `json:"fts_index_status"`
	AnalyticNodeDiag     *AnalyticsNodeDiagnostics `json:"analytics_node_diag"`
	AllServiceHosts      []string                  `json:"all_service_hosts"`
	UILogs               []UILogEntry              `json:"ui_logs"`
}

// Cache REST Data Errors. This struct will contain all the errors thrown from data retrieval
type CacheRESTDataErrors struct {
	BucketsError              error
	BucketStatsErrors         map[string]error
	ServerGroupsError         error
	NodeStorageError          error
	AutoFailoverSettingsError error
	IndexStatusError          error
	GSISettingsError          error
	IndexStorageStatsError    error
	FTSIndexStatusError       error
	AnalyticNodeDiagError     error
	AllServiceHostsError      error
	UILogsError               error
}

// CouchbaseCluster is the basic representation of a Couchbase Cluster in the manager. Note that the user and password
// will never be marshalled.
type CouchbaseCluster struct {
	UUID             string         `json:"uuid"`
	Name             string         `json:"name"`
	Alias            string         `json:"alias,omitempty"`
	User             string         `json:"-"`
	Password         string         `json:"-"`
	Enterprise       bool           `json:"enterprise"`
	NodesSummary     NodesSummary   `json:"nodes_summary"`
	BucketsSummary   BucketsSummary `json:"buckets_summary"`
	RemoteClusters   RemoteClusters `json:"remote_clusters"`
	ClusterInfo      *ClusterInfo   `json:"cluster_info"`
	PoolsRaw         []byte         `json:"-"`
	DeveloperPreview bool           `json:"developer_preview"`
	HeartBeatIssue   HeartIssue     `json:"heart_beat_issue,omitempty"`
	LastUpdate       time.Time      `json:"last_update"`
	CaCert           []byte         `json:"-"`

	StatusSummary  *ClusterStatusSummary `json:"status_summary,omitempty"`
	StatusProgress *ClusterProgress      `json:"status_progress,omitempty"`

	CacheRESTData       CacheRESTData       `json:"cache_rest_data,omitempty"`
	CacheRESTDataErrors CacheRESTDataErrors `json:"cache_rest_data_errors,omitempty"`
}

// GetTLSConfig returns a TLS config that has the CA if the cluster has an associated CA.
func (c *CouchbaseCluster) GetTLSConfig() *tls.Config {
	if !c.Enterprise {
		return nil
	}

	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	var ok bool
	if c.CaCert != nil {
		ok = rootCAs.AppendCertsFromPEM(c.CaCert)
	}

	return &tls.Config{InsecureSkipVerify: !ok, RootCAs: rootCAs}
}

// function to populate CacheRESTData within CouchbaseCluster
func (c *CacheRESTData) WithCacheRESTData(
	buckets []Bucket,
	bucketsStats []*BucketStat,
	serverGroups []ServerGroup,
	nodeStorage *Storage,
	autoFailOverSettings *AutoFailoverSettings,
	indexStatus []*IndexStatus,
	gsiSettings *GSISettings,
	indexStorageStats []*IndexStatsStorage,
	ftsIndexStatus FTSIndexStatus,
	analyticalNodeDiag *AnalyticsNodeDiagnostics,
	uiLogs []UILogEntry,
) *CacheRESTData {
	c.Buckets = buckets
	c.BucketStats = bucketsStats
	c.ServerGroups = serverGroups
	c.NodeStorage = nodeStorage
	c.AutoFailOverSettings = autoFailOverSettings
	c.IndexStatus = indexStatus
	c.GSISettings = gsiSettings
	c.IndexStorageStats = indexStorageStats
	c.FTSIndexStatus = ftsIndexStatus
	c.AnalyticNodeDiag = analyticalNodeDiag
	c.UILogs = uiLogs

	return c
}

// functions to retrieve data from within CacheRESTData
func (c *CouchbaseCluster) GetCacheBuckets() ([]Bucket, error) {
	return c.CacheRESTData.Buckets, c.CacheRESTDataErrors.BucketsError
}

func (c *CouchbaseCluster) GetCacheBucketStats() ([]*BucketStat, map[string]error) {
	return c.CacheRESTData.BucketStats, c.CacheRESTDataErrors.BucketStatsErrors
}

func (c *CouchbaseCluster) GetCacheServerGroups() ([]ServerGroup, error) {
	return c.CacheRESTData.ServerGroups, c.CacheRESTDataErrors.ServerGroupsError
}

func (c *CouchbaseCluster) GetCacheNodeStorage() (*Storage, error) {
	return c.CacheRESTData.NodeStorage, c.CacheRESTDataErrors.NodeStorageError
}

func (c *CouchbaseCluster) GetCacheAutoFailOverSettings() (*AutoFailoverSettings, error) {
	return c.CacheRESTData.AutoFailOverSettings,
		c.CacheRESTDataErrors.AutoFailoverSettingsError
}

func (c *CouchbaseCluster) GetCacheIndexStatus() ([]*IndexStatus, error) {
	return c.CacheRESTData.IndexStatus, c.CacheRESTDataErrors.IndexStatusError
}

func (c *CouchbaseCluster) GetCacheGSISettings() (*GSISettings, error) {
	return c.CacheRESTData.GSISettings, c.CacheRESTDataErrors.GSISettingsError
}

func (c *CouchbaseCluster) GetCacheIndexStorageStats() ([]*IndexStatsStorage, error) {
	return c.CacheRESTData.IndexStorageStats,
		c.CacheRESTDataErrors.IndexStatusError
}

func (c *CouchbaseCluster) GetCacheFTSIndexStatus() (FTSIndexStatus, error) {
	return c.CacheRESTData.FTSIndexStatus,
		c.CacheRESTDataErrors.FTSIndexStatusError
}

func (c *CouchbaseCluster) GetCacheAnalyticalNodeDiag() (*AnalyticsNodeDiagnostics, error) {
	return c.CacheRESTData.AnalyticNodeDiag, c.CacheRESTDataErrors.AnalyticNodeDiagError
}

func (c *CouchbaseCluster) GetCacheUILogs() ([]UILogEntry, error) {
	return c.CacheRESTData.UILogs, c.CacheRESTDataErrors.UILogsError
}

// NodesSummary a convenient alias for a slice of NodeSummaries.
type NodesSummary []NodeSummary

// GetHosts returns a slice with all the hosts in the cluster. They are all https and use the secure admin port.
func (c NodesSummary) GetHosts() []string {
	hosts := make([]string, len(c))
	for i, node := range c {
		hosts[i] = node.Host
	}

	return hosts
}

// Get Uptimes of individual nodes in the cluster. This will be used in MB-37643.
func (c NodesSummary) GetHighestUptime() (uint64, error) {
	lowest := uint64(0)
	for _, node := range c {
		uptime, err := strconv.ParseUint(node.Uptime, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("could not parse node(%v) uptime: %w", node.NodeUUID, err)
		}
		if uptime > lowest {
			lowest = uptime
		}
	}

	return lowest, nil
}

// GetMinVersion returns the oldest node version in the nodes summary.
func (c NodesSummary) GetMinVersion() cbvalue.Version {
	if len(c) == 0 {
		return cbvalue.VersionUnknown
	}

	minVersion := cbvalue.Version(c[0].Version)

	for _, node := range c[1:] {
		if cbvalue.Version(node.Version).Older(minVersion) {
			minVersion = cbvalue.Version(node.Version)
		}
	}

	return minVersion
}

// GetHostVersion returns a map of each host's version in NodesSummary
func (c NodesSummary) GetHostVersions() (map[string]cbvalue.Version, error) {
	hostVersion := make(map[string]cbvalue.Version)

	for _, node := range c {
		host, _, err := net.SplitHostPort(netutil.TrimSchema(node.Host))
		if err != nil {
			return nil, fmt.Errorf("could not split hostname and port")
		}
		hostVersion[host] = cbvalue.Version(node.Version)
	}

	return hostVersion, nil
}

// NodeSummary is the representation of a Couchbase Node. It contains some general node information.
type NodeSummary struct {
	NodeUUID          string   `json:"node_uuid"`
	Version           string   `json:"version,omitempty"`
	Host              string   `json:"host,omitempty"`
	OS                string   `json:"os,omitempty"`
	Status            string   `json:"status,omitempty"`
	ClusterMembership string   `json:"cluster_membership,omitempty"`
	Services          []string `json:"services,omitempty"`
	SwapUsed          uint64   `json:"swap_used,omitempty"`
	SwapTotal         uint64   `json:"swap_total,omitempty"`
	CPUUtil           float64  `json:"cpu_utilization_rate,omitempty"`
	MemTotal          uint64   `json:"mem_total,omitempty"`
	MemFree           uint64   `json:"mem_free,omitempty"`
	CPUCount          int      `json:"cpuCount,omitempty"`
	Uptime            string   `json:"uptime,omitempty"`
}

// HasService returns whether this node has the given service.
func (n NodeSummary) HasService(service string) bool {
	for _, svc := range n.Services {
		if service == svc {
			return true
		}
	}
	return false
}

// BucketsSummary is a convenient alias for a slice of BucketSummaries.
type BucketsSummary []BucketSummary

// GetBucketNames returns a slice with the names off al the buckets.
func (b BucketsSummary) GetBucketNames() []string {
	bucketNames := make([]string, len(b))
	for i, bucket := range b {
		bucketNames[i] = bucket.Name
	}

	return bucketNames
}

// GetBucket returns the summary for the given bucket, or nil if it does not exist.
func (b BucketsSummary) GetBucket(bucketName string) *BucketSummary {
	for i, bucket := range b {
		if bucket.Name == bucketName {
			return &b[i]
		}
	}
	return nil
}

// Functions used for sorting the bucket summary by bucket name.
func (b BucketsSummary) Len() int           { return len(b) }
func (b BucketsSummary) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b BucketsSummary) Less(i, j int) bool { return b[i].Name < b[j].Name }

// MarshallBucketsSummaryFromRest gets the bucket summary from the /pools/default/buckets endpoint. The resulting slice
// is sorted by bucket name to make sure that the buckets are always given in the same order.
func MarshallBucketsSummaryFromRest(body io.Reader) (BucketsSummary, error) {
	var overlay []struct {
		Name                   string `json:"name"`
		CompressionMode        string `json:"compressionMode"`
		ConflictResolutionType string `json:"conflictResolutionType"`
		BucketType             string `json:"bucketType"`
		StorageBackend         string `json:"storageBackend"`
		EvictionPolicy         string `json:"evictionPolicy"`
		NumReplicas            uint64 `json:"replicaNumber"`
		Quota                  struct {
			RAM uint64 `json:"ram"`
		} `json:"quota"`
		BasicStats struct {
			QuotaPercentUsed float64 `json:"quotaPercentUsed"`
			ItemCount        uint64  `json:"itemCount"`
		} `json:"basicStats"`
		Controllers struct {
			Flush string `json:"flush"`
		} `json:"controllers"`
	}

	err := json.NewDecoder(body).Decode(&overlay)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall data: %w", err)
	}

	buckets := make(BucketsSummary, len(overlay))
	for i, bucket := range overlay {
		buckets[i] = BucketSummary{
			Name:                   bucket.Name,
			CompressionMode:        bucket.CompressionMode,
			ConflictResolutionType: bucket.ConflictResolutionType,
			BucketType:             bucket.BucketType,
			StorageBackend:         bucket.StorageBackend,
			EvictionPolicy:         bucket.EvictionPolicy,
			Quota:                  bucket.Quota.RAM,
			QuotaUsed:              bucket.BasicStats.QuotaPercentUsed,
			FlushEnabled:           bucket.Controllers.Flush != "",
			NumReplicas:            bucket.NumReplicas,
			Items:                  bucket.BasicStats.ItemCount,
		}

		if buckets[i].BucketType == "membase" {
			buckets[i].BucketType = "couchbase"
		}
	}

	// provide stable output plus as at most there will be 30 elements it won't take long
	sort.Sort(buckets)
	return buckets, nil
}

// BucketSummary contains basic information about a Couchbase Server bucket.
type BucketSummary struct {
	Name                   string  `json:"name"`
	CompressionMode        string  `json:"compression_mode"`
	ConflictResolutionType string  `json:"conflict_resolution_type"`
	BucketType             string  `json:"bucket_type"`
	StorageBackend         string  `json:"storage_backend"`
	EvictionPolicy         string  `json:"eviction_policy"`
	Quota                  uint64  `json:"quota"`
	QuotaUsed              float64 `json:"quota_used"`
	FlushEnabled           bool    `json:"flush_enabled"`
	NumReplicas            uint64  `json:"num_replicas"`
	Items                  uint64  `json:"items"`
}

type BucketStats []BucketStat

type BucketStat struct {
	VbActiveRatio []float64 `json:"vb_active_resident_items_ratio"`
	MemUsed       []float64 `json:"mem_used"`
}

type NodeStorage []DiskStorage

type DiskStorage struct {
	Path       string `json:"path"`
	SizeKBytes uint64 `json:"sizeKbytes"`
	Usage      uint64 `json:"usagePercent"`
}

type AvailableStorage struct {
	DiskStorage []DiskStorage `json:"hdd"`
}

type StorageConfig struct {
	Path         string   `json:"path"`
	IndexPath    string   `json:"index_path"`
	CBASDirs     []string `json:"cbas_dirs"`
	EventingPath string   `json:"eventing_path"`
	JavaHome     string   `json:"java_home"`
}

func (s StorageConfig) GetPathForService(service string) []string {
	switch service {
	case "kv":
		if len(s.Path) > 0 {
			return []string{s.Path}
		}
	case "index":
		fallthrough
	case "fts":
		if len(s.IndexPath) > 0 {
			return []string{s.IndexPath}
		}
	case "eventing":
		if len(s.EventingPath) > 0 {
			return []string{s.EventingPath}
		}
	case "cbas":
		return s.CBASDirs
	}
	return nil
}

func (s StorageConfig) GetAllPaths() map[string][]string {
	result := make(map[string][]string)
	for _, svc := range []string{"kv", "index", "fts", "eventing", "cbas"} {
		if _, ok := result[svc]; !ok {
			result[svc] = make([]string, 0)
		}
		result[svc] = append(result[svc], s.GetPathForService(svc)...)
	}
	return result
}

type NodeStorageSet struct {
	SSD []StorageConfig `json:"ssd"`
	HDD []StorageConfig `json:"hdd"`
}

type Storage struct {
	Available   AvailableStorage `json:"availableStorage"`
	NodeStorage NodeStorageSet   `json:"storage"`
}
