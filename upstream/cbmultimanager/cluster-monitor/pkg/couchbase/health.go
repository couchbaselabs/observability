// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/couchbase/tools-common/netutil"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

// CMOS-267 tracks making this configurable.
const standaloneAgentPort = 9092

func (c *Client) getAgentStandalone(node values.NodeSummary) ([]byte, error) {
	host, _, _ := net.SplitHostPort(netutil.TrimSchema(node.Host))
	url := fmt.Sprintf("http://%s:%d/api/v1/checkers", host, standaloneAgentPort)
	zap.S().Debugw("(Health) Requesting standalone health checkers", "url", url)
	res, err := http.Get(url)
	if err != nil {
		zap.S().Debugw("(Health) Failed to get standalone", "err", err)
		// Check if this is a "connection refused" error, which could just be that the agent is not running
		// Can't use `errors.Is()` with `syscall.Errno`, because errno numbers are OS-dependent.
		if strings.HasSuffix(err.Error(), "connection refused") {
			return nil, values.ErrNotFound
		}
		return nil, fmt.Errorf("could not get node checkers: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	return body, nil
}

func (c *Client) getAgentIntegrated() ([]byte, error) {
	res, err := c.get(CheckersNodeEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get: %w", err)
	}
	return res.Body, nil
}

const useIntegratedAgent = false

func (c *Client) GetCheckers(node values.NodeSummary) (map[string]*values.WrappedCheckerResult, error) {
	var (
		body []byte
		err  error
	)

	if useIntegratedAgent {
		body, err = c.getAgentIntegrated()
		if errors.Is(err, values.ErrNotFound) {
			body, err = c.getAgentStandalone(node)
		}
	} else {
		body, err = c.getAgentStandalone(node)
	}
	if err != nil {
		return nil, err
	}

	var results map[string]*values.WrappedCheckerResult
	if err = json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("could not unmarshal the checker results: %w", err)
	}

	return results, nil
}
