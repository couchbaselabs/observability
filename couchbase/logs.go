package couchbase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/cbrest"
)

func (c *Client) GetUILogs() ([]UILogEntry, error) {
	res, err := c.get(UILogsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("could not get UI logs: %w", err)
	}

	var uiLogs UILogs
	if err = json.Unmarshal(res.Body, &uiLogs); err != nil {
		return nil, fmt.Errorf("could not unmarshal UI logs: %w", err)
	}

	return uiLogs.List, err
}

// GetSASLLogs returns the response body reader. It is the callers responsibility to close the body.
func (c *Client) GetSASLLogs(ctx context.Context, logName string) (io.ReadCloser, error) {
	return c.GetLogREST(ctx, SASLLogsEndpoint.Format(logName))
}

// GetDiagLog returns the response body reader. It is the callers responsibility to close the body.
func (c *Client) GetDiagLog(ctx context.Context) (io.ReadCloser, error) {
	return c.GetLogREST(ctx, "/diag")
}

// GetLogREST returns the response body reader. It is the callers responsibility to close the body.
func (c *Client) GetLogREST(ctx context.Context, endpoint cbrest.Endpoint) (io.ReadCloser, error) {
	res, err := c.internalClient.Do(ctx, &cbrest.Request{
		Method:             http.MethodGet,
		Endpoint:           endpoint,
		ExpectedStatusCode: http.StatusOK,
		Service:            cbrest.ServiceManagement,
		ContentType:        "text/plain; charset=utf-8",
	})
	if err != nil {
		return nil, fmt.Errorf("could not get logs from cluster: %w", err)
	}

	switch res.StatusCode {
	case http.StatusOK:
		return res.Body, nil
	case http.StatusNotFound:
		_ = res.Body.Close()
		return nil, values.ErrNotFound
	}

	_ = res.Body.Close()
	return nil, fmt.Errorf("unexpected status code %d", res.StatusCode)
}
