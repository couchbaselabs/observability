// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package memcached

import (
	"fmt"

	memcached "github.com/couchbase/gomemcached/client"
	"github.com/couchbase/tools-common/errdefs"
)

// getStats is reused by any function that needs to make a cbstats call.
func (m *MemDClient) getStats(key string, bucket string) (map[string][]memcached.StatValue, error) {
	stats := make(map[string][]memcached.StatValue)
	errs := &errdefs.MultiError{
		Prefix: "failed to get stats for some nodes: ",
	}

	for _, host := range m.manager.Hosts() {
		err := m.manager.ClientForNode(host, func(client *memcached.Client) error {
			_, err := client.SelectBucket(bucket)
			if err != nil {
				return fmt.Errorf("could not select bucket: %w", err)
			}

			stat, err := client.Stats(key)
			if err != nil {
				return fmt.Errorf("could not collect memcached stats: %w", err)
			}

			if stat == nil {
				return fmt.Errorf("no %s stats available: %w", key, err)
			}

			stats[host] = stat
			return nil
		})
		if err != nil {
			errs.Add(fmt.Errorf("failed to collect stats from %s: %w", host, err))
		}
	}
	return stats, errs.ErrOrNil()
}
