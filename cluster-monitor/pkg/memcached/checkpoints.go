// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package memcached

import (
	"fmt"
	"strconv"
	"strings"

	memcached "github.com/couchbase/gomemcached/client"
)

func parseCheckpointStats(stats []memcached.StatValue) (BucketCheckpointStats, error) {
	// We're using a map instead of a slice because memcached returns the vBuckets in ascii-alphabetical order
	// (i.e. vb 1000 will come before vb 2), meaning we don't actually know the final number of vBs until we're done.
	// Allocate for 1024 because in 99% of cases we will need 1024.
	temp := make(map[int]map[string]string, 1024)
	for _, stat := range stats {
		parts := strings.SplitN(stat.Key, ":", 2)
		// For checkpoint stats, the first key will always be in the format vb_xxxx
		vbNoStr := parts[0][3:]
		vbNo, err := strconv.Atoi(vbNoStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing checkpoint stats: %w", err)
		}
		statData, ok := temp[vbNo]
		if !ok {
			temp[vbNo] = make(map[string]string)
			statData = temp[vbNo]
		}
		statData[parts[1]] = stat.Val
	}

	maxVB := 0
	for vb := range temp {
		if maxVB < vb {
			maxVB = vb
		}
	}

	result := make(BucketCheckpointStats, maxVB+1)
	for vb, stats := range temp {
		result[vb] = stats
	}
	return result, nil
}

// CheckpointStats returns statistics about Memcached checkpoints for the given bucket and host.
// It is the equivalent of running `cbstats checkpoint`.
func (m *MemDClient) CheckpointStats(host, bucket string) (BucketCheckpointStats, error) {
	var stats []memcached.StatValue
	err := m.manager.ClientForNode(host, func(client *memcached.Client) error {
		_, err := client.SelectBucket(bucket)
		if err != nil {
			return fmt.Errorf("could not select bucket: %w", err)
		}

		stats, err = client.Stats("checkpoint")
		if err != nil {
			return fmt.Errorf("could not get checkpoint stats from memcached: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return parseCheckpointStats(stats)
}
