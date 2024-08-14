// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

// Adapted from https://github.com/couchbase/indexing/blob/cheshire-cat/secondary/manager/request_handler.go

type IndexStatus struct {
	DefnID uint64 `json:"defnId,omitempty"`
	InstID uint64 `json:"instId,omitempty"`
	// TODO: according to these type defs (taken from the indexing source), the index name is stored in `name`,
	// but in my testing it actually get stored in `index`. Is ns_server renaming the field somewhere?
	Name         string           `json:"index,omitempty"`
	Bucket       string           `json:"bucket,omitempty"`
	Scope        string           `json:"scope,omitempty"`
	Collection   string           `json:"collection,omitempty"`
	IsPrimary    bool             `json:"isPrimary,omitempty"`
	IndexType    string           `json:"indexType,omitempty"`
	Status       string           `json:"status,omitempty"`
	Definition   string           `json:"definition"`
	Hosts        []string         `json:"hosts,omitempty"`
	Scheduled    bool             `json:"scheduled"`
	PartitionMap map[string][]int `json:"partitionMap,omitempty"`
	Partitioned  bool             `json:"partitioned"`
	NumPartition int              `json:"numPartition"`

	NumReplica   int    `json:"numReplica"`
	IndexName    string `json:"indexName"`
	ReplicaID    int    `json:"replicaId"`
	Stale        bool   `json:"stale"`
	LastScanTime string `json:"lastScanTime,omitempty"`
}

type IndexStatsStorage struct {
	Name        string       `json:"Index"`
	PartitionID int          `json:"PartitionId"`
	Stats       GSIMainStore `json:"Stats"`
}

type GSIMainStore struct {
	GSIStore `json:"MainStore"`
}

type GSIStore struct {
	IndexMemory int `json:"memory_size_index"`
}
