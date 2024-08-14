// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/couchbase/tools-common/cbrest"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func (c *Client) GetFTSIndexStatus() (values.FTSIndexStatus, error) {
	// getFTSIndexStatus is a scatter-gather endpoint, so we only need to make the request to one FTS node
	res, err := c.internalClient.Execute(&cbrest.Request{
		Method:             http.MethodGet,
		Endpoint:           "/api/index",
		Service:            cbrest.ServiceSearch,
		ExpectedStatusCode: http.StatusOK,
	})
	if err != nil {
		return values.FTSIndexStatus{}, fmt.Errorf("could not get FTS index status: %w", err)
	}

	var result values.FTSIndexStatus
	if err = json.Unmarshal(res.Body, &result); err != nil {
		return values.FTSIndexStatus{}, fmt.Errorf("could not unmarshal FTS index status: %w", err)
	}

	return result, nil
}
