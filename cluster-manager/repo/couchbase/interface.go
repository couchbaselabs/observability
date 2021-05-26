package couchbase

import (
	"context"
	"io"
	"time"

	"github.com/couchbaselabs/cbmultimanager/values"
)

// ClientIFace is an interface that will be used to switch a real REST client for a test one during unit testing.
type ClientIFace interface {
	GetPoolsBucket() ([]Bucket, error)
	GetBucketsSummary() (values.BucketsSummary, error)
	GetBucketStats(bucketName string) (*values.BucketStat, error)
	GetAutoFailOverSettings() (*AutoFailoverSettings, error)
	GetUILogs() ([]UILogEntry, error)
	GetSASLLogs(ctx context.Context, logName string) (io.ReadCloser, error)
	GetDiagLog(ctx context.Context) (io.ReadCloser, error)
	GetNodesSummary() (values.NodesSummary, error)
	GetMetric(start, end, metricName, step string) (*Metric, error)
	GetNodeStorage() (*values.Storage, error)

	GetBootstrap() time.Time
	GetClusterInfo() *PoolsMetadata
}
