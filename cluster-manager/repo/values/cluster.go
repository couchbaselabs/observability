package values

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	// Blank embed for versionsfile
	_ "embed"

	"go.uber.org/zap"
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

// CouchbaseCluster is the basic representation of a Couchbase Cluster in the manager. Note that the user and password
// will never be marshalled.
type CouchbaseCluster struct {
	UUID           string         `json:"uuid"`
	Name           string         `json:"name"`
	User           string         `json:"-"`
	Password       string         `json:"-"`
	NodesSummary   NodesSummary   `json:"nodes_summary"`
	BucketsSummary BucketsSummary `json:"buckets_summary"`
	ClusterInfo    *ClusterInfo   `json:"cluster_info"`
	HeartBeatIssue HeartIssue     `json:"heart_beat_issue,omitempty"`
	LastUpdate     time.Time      `json:"last_update"`
	CaCert         []byte         `json:"-"`

	StatusSummary  *ClusterStatusSummary `json:"status_summary,omitempty"`
	StatusProgress *ClusterProgress      `json:"status_progress"`
}

// GetTlSConfig returns a TLS config that has the CA if the cluster has an associated CA.
func (c *CouchbaseCluster) GetTLSConfig() *tls.Config {
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

// NodeSummary is the representation of a Couchbase Node. It contains some general node information.
type NodeSummary struct {
	NodeUUID          string   `json:"node_uuid"`
	Version           string   `json:"version,omitempty"`
	Host              string   `json:"host,omitempty"`
	Status            string   `json:"status,omitempty"`
	ClusterMembership string   `json:"cluster_membership,omitempty"`
	Services          []string `json:"services,omitempty"`
	SwapUsed          uint64   `json:"swap_used,omitempty"`
	SwapTotal         uint64   `json:"swap_total,omitempty"`
	CPUUtil           float64  `json:"cpu_utilization_rate,omitempty"`
	MemTotal          uint64   `json:"mem_total,omitempty"`
	MemFree           uint64   `json:"mem_free,omitempty"`
	CPUCount          int      `json:"cpuCount,omitempty"`
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

type Versions map[string]Version

type Version struct {
	Build string    `json:"version"`
	EOM   time.Time `json:"-"`
	EOS   time.Time `json:"-"`
	OS    []string  `json:"OS"`
}

//go:embed versions.json
var verByte []byte

// GAVersions holds the parsed version.json information file
var GAVersions Versions

func init() {
	var err error
	GAVersions, err = getVersions()
	if err != nil {
		GAVersions = make(Versions)
		zap.S().Errorw("(Values) Could not load GA versions", "err", err)
	}
}

// GetVersions reads the versions file and returns the parsed output
func getVersions() (Versions, error) {
	versionList, err := parseVersions(verByte)
	if err != nil {
		return nil, fmt.Errorf("could not parse versions: %w", err)
	}

	return versionList, nil
}

// UnmarshalJSON unmarshals Versions to include time.Time in the correct format
func (v *Version) UnmarshalJSON(data []byte) (err error) {
	type timeOverlay struct {
		Build string   `json:"version"`
		OS    []string `json:"OS"`
		EOM   string   `json:"EOM"`
		EOS   string   `json:"EOS"`
	}

	var timeData timeOverlay
	if err := json.Unmarshal(data, &timeData); err != nil {
		return err
	}

	v.Build = timeData.Build
	v.OS = timeData.OS
	v.EOM, err = time.Parse("2006-01-02", timeData.EOM)
	if err != nil {
		return err
	}

	v.EOS, err = time.Parse("2006-01-02", timeData.EOS)
	return err
}

// parseVersions parses the versions file
func parseVersions(byteval []byte) (Versions, error) {
	var overlay struct {
		Versions []Version `json:"versions"`
	}

	err := json.Unmarshal(byteval, &overlay)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal version data %w", err)
	}

	versionList := make(map[string]Version)
	for _, ver := range overlay.Versions {
		versionList[ver.Build] = ver
	}

	return versionList, nil
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

type Storage struct {
	Available AvailableStorage `json:"availableStorage"`
}
