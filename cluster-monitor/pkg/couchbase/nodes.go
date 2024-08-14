// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/netutil"
	"go.uber.org/zap"
)

// GetNodesSummary returns a slice with the addresses of all the nodes in the cluster
func (c *Client) GetNodesSummary() (values.NodesSummary, error) {
	res, err := c.get(PoolsNodesEndpoint)
	if err != nil {
		return nil, fmt.Errorf("could not get node summary: %w", err)
	}

	type node struct {
		NodeUUID           string            `json:"nodeUUID"`
		Hostname           string            `json:"hostname"`
		Services           []string          `json:"services"`
		Version            string            `json:"version"`
		Status             string            `json:"status"`
		ClusterMembership  string            `json:"clusterMembership"`
		Ports              map[string]uint16 `json:"ports"`
		CPUCount           json.RawMessage   `json:"cpuCount"`
		OS                 string            `json:"os"`
		AlternateAddresses *struct {
			External *AlternateAddresses `json:"external"`
		} `json:"alternateAddresses"`
		SystemStats struct {
			CPUUtil   float64 `json:"cpu_utilization_rate"`
			SwapTotal uint64  `json:"swap_total"`
			SwapUsed  uint64  `json:"swap_used"`
			MemTotal  uint64  `json:"mem_total"`
			MemFree   uint64  `json:"mem_free"`
		} `json:"systemStats"`
		Uptime string `json:"uptime"`
	}

	type overlay struct {
		Nodes []node `json:"nodes"`
	}

	var nodesData overlay
	if err := json.Unmarshal(res.Body, &nodesData); err != nil {
		return nil, fmt.Errorf("could not unmarshal server response: %w", err)
	}

	useAlt := c.internalClient.AltAddr()
	summary := make([]values.NodeSummary, 0, len(nodesData.Nodes))

	getAltPort := func(node node) (uint16, error) {
		if c.ClusterInfo.Enterprise {
			if node.AlternateAddresses.External.Services.ManagementSSL != 0 {
				return node.AlternateAddresses.External.Services.ManagementSSL, nil
			}

			if node.AlternateAddresses.External.Services.CapiSSL != 0 {
				return node.AlternateAddresses.External.Services.CapiSSL, nil
			}

			return node.Ports["httpsMgmt"], nil
		}

		if node.AlternateAddresses.External.Services.Management != 0 {
			return node.AlternateAddresses.External.Services.Management, nil
		}

		if node.AlternateAddresses.External.Services.Capi != 0 {
			return node.AlternateAddresses.External.Services.Capi, nil
		}

		_, portStr, err := net.SplitHostPort(node.Hostname)
		if err != nil {
			return 0, fmt.Errorf("hostname '%s' does not have have port: %w", node.Hostname, err)
		}

		portInt, err := strconv.Atoi(portStr)
		if err != nil {
			return 0, fmt.Errorf("invalid port '%s'", portStr)
		}

		return uint16(portInt), nil
	}

	getNodeHostName := func(useAlt bool, node node) (string, error) {
		if useAlt {
			scheme := "http://"
			if c.ClusterInfo.Enterprise {
				scheme = "https://"
			}

			if node.AlternateAddresses.External.Hostname == "" {
				node.AlternateAddresses.External.Hostname = "localhost"
			}

			port, err := getAltPort(node)
			if err != nil {
				return "", fmt.Errorf("could not get alternate address management port: %w", err)
			}

			return fmt.Sprintf("%s%s:%d", scheme, netutil.ReconstructIPV6(node.AlternateAddresses.External.Hostname),
				port), nil
		}

		if !c.ClusterInfo.Enterprise {
			return fmt.Sprintf("http://%s", node.Hostname), nil
		}

		nodeHost, _, err := net.SplitHostPort(node.Hostname)
		if err != nil {
			// in theory this should never happen but we still check just in case
			return "", fmt.Errorf("hostname '%s' does not have have port: %w", node.Hostname,
				err)
		}

		return fmt.Sprintf("https://%s:%d", netutil.ReconstructIPV6(nodeHost), node.Ports["httpsMgmt"]), nil
	}

	for _, node := range nodesData.Nodes {
		// In some cases the CPUCount can be set to "unknown" which has to be handled correctly.
		var cpuCount int
		if err := json.Unmarshal(node.CPUCount, &cpuCount); err != nil {
			zap.S().Warnw("(Couchbase) Received invalid CPU count from nodes endpoint", "count", string(node.CPUCount),
				"node", node.Hostname)
		}

		nodeSummary := values.NodeSummary{
			Version:           node.Version,
			Status:            node.Status,
			NodeUUID:          node.NodeUUID,
			ClusterMembership: node.ClusterMembership,
			Services:          node.Services,
			OS:                node.OS,
			CPUCount:          cpuCount,
			CPUUtil:           node.SystemStats.CPUUtil,
			SwapUsed:          node.SystemStats.SwapUsed,
			SwapTotal:         node.SystemStats.SwapTotal,
			MemFree:           node.SystemStats.MemFree,
			MemTotal:          node.SystemStats.MemTotal,
			Uptime:            node.Uptime,
		}

		nodeSummary.Host, err = getNodeHostName(useAlt, node)
		if err != nil {
			return nil, err
		}

		// Some versions do not expose the node uuid in that case we will use the hostname as the node uuid. Note this
		// value can change in some specific cases (when changing a 1 node cluster two a 2 node cluster) but it is the
		// best we can do.
		if nodeSummary.NodeUUID == "" {
			nodeSummary.NodeUUID = nodeSummary.Host
		}

		summary = append(summary, nodeSummary)
	}

	return summary, nil
}

func (c *Client) GetAllServiceHosts(service cbrest.Service) ([]string, error) {
	return c.internalClient.GetAllServiceHosts(service)
}

func (c *Client) GetNodeStorage() (*values.Storage, error) {
	res, err := c.get(NodesSelfEndpoint)
	if err != nil {
		return nil, fmt.Errorf("could not issue get request: %w", err)
	}

	var storage values.Storage
	err = json.Unmarshal(res.Body, &storage)

	if err != nil {
		return nil, fmt.Errorf("could not unmarshal storage information: %w", err)
	}

	return &storage, nil
}

// PingService verifies that this cbmultimanager can communicate with the given service.
func (c *Client) PingService(service cbrest.Service) error {
	var endpoint cbrest.Endpoint
	switch service {
	case cbrest.ServiceQuery, cbrest.ServiceAnalytics:
		endpoint = "/admin/ping"
	case cbrest.ServiceSearch:
		endpoint = "/api/ping"
	case cbrest.ServiceManagement:
		endpoint = "/pools/default/terseClusterInfo"
	case cbrest.ServiceViews:
		endpoint = "/"
	default:
		return fmt.Errorf("cannot ping %s with PingService", service)
	}
	_, err := c.internalClient.Execute(&cbrest.Request{
		Method:             http.MethodGet,
		Endpoint:           endpoint,
		Service:            service,
		ExpectedStatusCode: http.StatusOK,
	})
	if err == nil {
		return nil
	}

	var notFound *cbrest.EndpointNotFoundError
	if errors.As(err, &notFound) {
		return values.ErrNotFound
	}

	return getAuthError(err)
}

func (c *Client) GetAnalyticsNodeDiagnostics() (*values.AnalyticsNodeDiagnostics, error) {
	res, err := c.internalClient.Execute(&cbrest.Request{
		Method:             http.MethodGet,
		Endpoint:           AnalyticsNodeDiagnosticEndpoint,
		Service:            cbrest.ServiceAnalytics,
		ExpectedStatusCode: http.StatusOK,
	})
	if err != nil {
		return nil, err
	}
	result := &values.AnalyticsNodeDiagnostics{}

	err = json.Unmarshal(res.Body, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
