// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"crypto/tls"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

type PoolsMetadata struct {
	Enterprise       bool                `json:"enterprise"`
	DeveloperPreview bool                `json:"developer_preview"`
	ClusterUUID      string              `json:"uuid"`
	ClusterName      string              `json:"name"`
	NodesSummary     values.NodesSummary `json:"nodes_summary"`
	ClusterInfo      *values.ClusterInfo `json:"cluster_info"`
	PoolsRaw         []byte              `json:"-"`
}

type AlternateAddresses struct {
	Hostname string    `json:"hostname"`
	Services *Services `json:"ports"`
}

type Services struct {
	Capi              uint16 `json:"capi"`
	CapiSSL           uint16 `json:"capiSSL"`
	Management        uint16 `json:"mgmt"`
	ManagementSSL     uint16 `json:"mgmtSSL"`
	FullText          uint16 `json:"fts"`
	FullTextSSL       uint16 `json:"ftsSSL"`
	SecondaryIndex    uint16 `json:"indexHttp"`
	SecondaryIndexSSL uint16 `json:"indexHttps"`
	N1QL              uint16 `json:"n1ql"`
	N1QLSSL           uint16 `json:"n1qlSSL"`
	Eventing          uint16 `json:"eventingAdminPort"`
	EventingSSL       uint16 `json:"eventingSSL"`
	Cbas              uint16 `json:"cbas"`
	CbasSSL           uint16 `json:"cbasSSL"`
	Backup            uint16 `json:"backupAPI"`
	BackupSSL         uint16 `json:"backupAPIHTTPS"`

	// non-services REST API endpoints
	KV    uint16 `json:"kv"`
	KVSSL uint16 `json:"kvSSL"`

	IndexAdmin         uint16 `json:"indexAdmin"`
	IndexScan          uint16 `json:"indexScan"`
	IndexStreamInit    uint16 `json:"indexStreamInit"`
	IndexStreamCatchup uint16 `json:"indexStreamCatchup"`
	IndexStreamMaint   uint16 `json:"indexStreamMaint"`

	FullTextGRPC    uint16 `json:"ftsGRPC"`
	FullTextGRPCSSL uint16 `json:"ftsGRPCSSL"`
}

// Metric contains basic metric information, alongside needed labels.
type Metric struct {
	Name     string      `json:"name"`
	Category string      `json:"system"`
	Instance string      `json:"instance,omitempty"`
	Job      string      `json:"job,omitempty"`
	Values   []MetricVal `json:"values"`
	Repo     string      `json:"repository,omitempty"`
}

type MetricVal struct {
	Timestamp time.Time `json:"timestamp"`
	Value     string    `json:"value"`
}

type clientAuth struct {
	tlsConfig *tls.Config
	username  string
	password  string
}

type VBucketServerMap struct {
	ServerList  []string `json:"serverList"`
	NumReplicas int      `json:"numReplicas"`
	VBucketMap  [][]int  `json:"vBucketMap"`
}
