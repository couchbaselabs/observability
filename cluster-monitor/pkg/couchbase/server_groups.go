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

type ServerGroupResult struct {
	Groups []values.ServerGroup `json:"groups"`
}

func (c *Client) GetServerGroups() ([]values.ServerGroup, error) {
	res, err := c.get(PoolsServerGroup)
	if err != nil {
		return nil, fmt.Errorf("could not get pools server group data: %w", err)
	}

	var serverGroups ServerGroupResult
	if err = json.Unmarshal(res.Body, &serverGroups); err != nil {
		return nil, fmt.Errorf("could not unmarshal server groups: %w", err)
	}

	return serverGroups.Groups, nil
}
