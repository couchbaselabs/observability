// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"context"
	"io"
	"time"

	"github.com/couchbase/tools-common/cbrest"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

//go:generate mockery --name ClientIFace

// ClientIFace is an interface that will be used to switch a real REST client for a test one during unit testing.
type ClientIFace interface {
	GetPoolsBucket() ([]values.Bucket, error)
	GetBucketsSummary() (values.BucketsSummary, error)
	GetBucketStats(bucketName string) (*values.BucketStat, error)
	GetAutoFailOverSettings() (*values.AutoFailoverSettings, error)
	GetUILogs() ([]values.UILogEntry, error)
	GetSASLLogs(ctx context.Context, logName string) (io.ReadCloser, error)
	GetDiagLog(ctx context.Context) (io.ReadCloser, error)
	GetNodesSummary() (values.NodesSummary, error)
	GetMetric(start, end, metricName, step string) (*Metric, error)
	GetNodeStorage() (*values.Storage, error)
	GetIndexStatus() ([]*values.IndexStatus, error)
	GetFTSIndexStatus() (values.FTSIndexStatus, error)
	PingService(service cbrest.Service) error
	GetBootstrap() time.Time
	GetClusterInfo() *PoolsMetadata
	GetServerGroups() ([]values.ServerGroup, error)
	GetIndexStorageStats() ([]*values.IndexStatsStorage, error)
}
