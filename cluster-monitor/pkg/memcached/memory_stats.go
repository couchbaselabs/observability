// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package memcached

import (
	"fmt"

	memcached "github.com/couchbase/gomemcached/client"
)

// ArenaFragmentationBytes and ArenaResidentBytes are the stats given by 7.x
// FragmentationBytes and HeapBytes are the stats given by 6.x
type MemoryStats struct {
	ArenaFragmentationBytes string `json:"ep_arena:fragmentation_size"`
	ArenaResidentBytes      string `json:"ep_arena:resident"`
	Host                    string `json:"host"`
	FragmentationBytes      string `json:"total_fragmentation_bytes"`
	HeapBytes               string `json:"total_heap_bytes"`
}

// MemStats collects memory stats for a given bucket from all nodes in the cluster.
func (m *MemDClient) MemStats(bucket string) ([]*MemoryStats, error) {
	stats := make([]*MemoryStats, 0, len(m.manager.Hosts()))

	statsRaw, err := m.getStats("memory", bucket)
	if err != nil {
		return nil, fmt.Errorf("could not collect memcached stats: %w", err)
	}

	for host, stat := range statsRaw {
		parsedStat := parseMemStatValue(stat)
		parsedStat.Host = host
		stats = append(stats, parsedStat)
	}

	return stats, nil
}

// parseMemStatValue provides a simpler output for statistics.
func parseMemStatValue(in []memcached.StatValue) *MemoryStats {
	var overlay MemoryStats
	for _, item := range in {
		switch item.Key {
		case "ep_arena:fragmentation_size":
			overlay.ArenaFragmentationBytes = item.Val
		case "ep_arena:resident":
			overlay.ArenaResidentBytes = item.Val
		case "total_fragmentation_bytes":
			overlay.FragmentationBytes = item.Val
		case "total_heap_bytes":
			overlay.HeapBytes = item.Val
		}
	}
	return &overlay
}
