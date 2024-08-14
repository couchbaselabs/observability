// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package memcached

import (
	"fmt"
	"regexp"

	memcached "github.com/couchbase/gomemcached/client"
	"go.uber.org/zap"
)

// ReplicationStat holds information on dcpq producer statistics.
type ReplicationStat struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Value       string `json:"value"`
	Extras      string `json:"extras"`
}

// DCPMemStats holds information about a data nodes memcached statistics.
type DCPMemStats struct {
	Count          string            `json:"ep_dcp_count"`
	TotalBytes     string            `json:"ep_dcp_total_bytes"`
	TotalQueue     string            `json:"ep_dcp_total_queue"`
	ItemsSent      string            `json:"ep_dcp_items_sent"`
	ItemsRemaining string            `json:"ep_dcp_items_remaining"`
	Host           string            `json:"host"`
	MaxBufferBytes []ReplicationStat `json:"max_buffer_bytes"`
	UnackedBytes   []ReplicationStat `json:"unacked_bytes"`
	PausedReason   []ReplicationStat `json:"paused_reason"`
	StreamNames    []string          `json:"stream_names"`
}

var (
	// NB: DCP stream names can contain colons
	dcpSingleConnectionStatRegex  = regexp.MustCompile(`^eq_dcpq:(?P<type>[^:]+):(?P<name>.+?):(?P<stat>[^:]*)$`)
	dcpReplicationStreamNameRegex = regexp.MustCompile(`^(?P<src>.+?)->(?P<dest>.+?):(?P<bucket>.+)$`)
)

// checkDCPReplicationStat checks dcp replication statistics.
func checkDcpReplicationStat(statName, streamName, value string, overlay DCPMemStats) DCPMemStats {
	// Format of a replication stream name is ns_1@host->ns_1@host:bucket
	// For example: ns_1@10.240.0.13->ns_1@10.240.0.6:travel-sample
	parts := dcpReplicationStreamNameRegex.FindStringSubmatch(streamName)
	if parts == nil {
		zap.S().Errorw("(Memcached) Malformed replication stream name", "streamName", streamName)
		return overlay
	}
	source := parts[dcpReplicationStreamNameRegex.SubexpIndex("src")]
	destination := parts[dcpReplicationStreamNameRegex.SubexpIndex("dest")]

	if statName == "max_buffer_bytes" {
		overlay.MaxBufferBytes = append(overlay.MaxBufferBytes, ReplicationStat{
			Name:        statName,
			Source:      source,
			Destination: destination,
			Value:       value,
		})
	}

	if statName == "unacked_bytes" {
		overlay.UnackedBytes = append(overlay.UnackedBytes, ReplicationStat{
			Name:        statName,
			Source:      source,
			Destination: destination,
			Value:       value,
		})
	}

	if statName == "paused_reason" {
		overlay.PausedReason = append(overlay.PausedReason, ReplicationStat{
			Name:        statName,
			Source:      source,
			Destination: destination,
			Value:       value,
		})
	}

	return overlay
}

// checkDCPStat checks basic statistics from a memcachedStat.
func checkDCPStat(name, value string, overlay DCPMemStats) DCPMemStats {
	switch name {
	case "ep_dcp_count":
		overlay.Count = value
	case "ep_dcp_total_bytes":
		overlay.TotalBytes = value
	case "ep_dcp_total_queue":
		overlay.TotalQueue = value
	case "ep_dcp_items_remaining":
		overlay.ItemsRemaining = value
	case "ep_dcp_items_sent":
		overlay.ItemsSent = value
	}

	return overlay
}

// parseDCPStatValues provides a simpler output for statistics.
func parseDCPStatValues(in []memcached.StatValue) *DCPMemStats {
	var overlay DCPMemStats
	streamNames := make(map[string]struct{})

	for _, item := range in {
		parts := dcpSingleConnectionStatRegex.FindStringSubmatch(item.Key)
		if parts == nil {
			overlay = checkDCPStat(item.Key, item.Val, overlay)
			continue
		}

		// eq_dcpq: doesn't count as part of the stream name for the purposes of MB-34280, neither does the stat,
		// but the type (`replication` etc.) does.
		fullStreamName := fmt.Sprintf(
			"%s:%s",
			parts[dcpSingleConnectionStatRegex.SubexpIndex("type")],
			parts[dcpSingleConnectionStatRegex.SubexpIndex("name")],
		)
		streamNames[fullStreamName] = struct{}{}
		streamType := parts[dcpSingleConnectionStatRegex.SubexpIndex("type")]
		// check if this is a Replication Stat
		if streamType == "replication" {
			statName := parts[dcpSingleConnectionStatRegex.SubexpIndex("stat")]
			overlay = checkDcpReplicationStat(
				statName,
				parts[dcpSingleConnectionStatRegex.SubexpIndex("name")],
				item.Val,
				overlay,
			)
		} else {
			overlay = checkDCPStat(item.Key, item.Val, overlay)
		}
	}

	for name := range streamNames {
		overlay.StreamNames = append(overlay.StreamNames, name)
	}

	return &overlay
}

// DCPStats collects DCP stats for a given bucket from all nodes in the cluster.
func (m *MemDClient) DCPStats(bucket string) ([]*DCPMemStats, error) {
	stats := make([]*DCPMemStats, 0, len(m.manager.Hosts()))

	statsRaw, err := m.getStats("dcp", bucket)
	if err != nil {
		return nil, fmt.Errorf("could not collect memcached stats: %w", err)
	}

	for host, stat := range statsRaw {
		parsedStat := parseDCPStatValues(stat)
		parsedStat.Host = host
		stats = append(stats, parsedStat)
	}

	return stats, nil
}
