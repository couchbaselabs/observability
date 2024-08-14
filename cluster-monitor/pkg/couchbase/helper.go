// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"errors"
	"net/http"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/cbrest"
)

func (c *Client) get(Endpoint cbrest.Endpoint) (*cbrest.Response, error) {
	res, err := c.internalClient.Execute(&cbrest.Request{
		Method:             http.MethodGet,
		Endpoint:           Endpoint,
		Service:            cbrest.ServiceManagement,
		ExpectedStatusCode: http.StatusOK,
	})
	if err == nil {
		return res, nil
	}

	var notFound *cbrest.EndpointNotFoundError
	if errors.As(err, &notFound) {
		return nil, values.ErrNotFound
	}

	return res, getAuthError(err)
}
