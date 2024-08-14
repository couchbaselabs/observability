// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"encoding/json"
	"fmt"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func (c *Client) GetRemoteClusters(fromName string, fromUUID string) (values.RemoteClusters, error) {
	res, err := c.get(PoolsRemoteCluster)
	if err != nil {
		return nil, fmt.Errorf("could not get pools remote clusters data: %w", err)
	}

	var remoteClusters values.RemoteClusters
	if err = json.Unmarshal(res.Body, &remoteClusters); err != nil {
		return nil, fmt.Errorf("could not unmarshal remote clusters: %w", err)
	}

	for index := range remoteClusters {
		remoteClusters[index].FromName = fromName
		remoteClusters[index].FromUUID = fromUUID
	}

	return remoteClusters, err
}
