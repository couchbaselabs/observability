// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package memcached

import (
	"fmt"
)

type DefStats struct {
	VbActiveSyncAccepted string `json:"vb_active_sync_write_accepted_count"`
	CmdGet               string `json:"cmd_get"`
	CmdSet               string `json:"cmd_set"`
	Host                 string `json:"host"`
}

// DefaultStats collects stats from the default cbstats group from all nodes in the cluster.
func (m *MemDClient) DefaultStats(bucket string) ([]*DefStats, error) {
	stats := make([]*DefStats, 0, len(m.manager.Hosts()))

	statsRaw, err := m.getStats("", bucket)
	if err != nil {
		return nil, fmt.Errorf("could not collect memcached stats: %w", err)
	}

	for host, stat := range statsRaw {
		var overlay DefStats
		for _, item := range stat {
			switch item.Key {
			case "vb_active_sync_write_accepted_count":
				overlay.VbActiveSyncAccepted = item.Val
			case "cmd_get":
				overlay.CmdGet = item.Val
			case "cmd_set":
				overlay.CmdSet = item.Val
			}
		}
		overlay.Host = host
		stats = append(stats, &overlay)
	}

	return stats, nil
}
