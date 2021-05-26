package couchbase

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/netutil"
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
		CPUCount           int               `json:"cpuCount"`
		AlternateAddresses *struct {
			External *AlternateAddresses `json:"external"`
		} `json:"alternate_addresses"`
		SystemStats struct {
			CPUUtil   float64 `json:"cpu_utilization_rate"`
			SwapTotal uint64  `json:"swap_total"`
			SwapUsed  uint64  `json:"swap_used"`
			MemTotal  uint64  `json:"mem_total"`
			MemFree   uint64  `json:"mem_free"`
		} `json:"systemStats"`
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

	getNodeHostName := func(useAlt bool, node node) (string, error) {
		if useAlt {
			if node.AlternateAddresses.External.Hostname == "" {
				node.AlternateAddresses.External.Hostname = "localhost"
			}

			// for alternate address the ports are not always populated
			port := node.Ports["httpsMgmt"]
			if node.AlternateAddresses.External.Services.ManagementSSL != 0 {
				port = node.AlternateAddresses.External.Services.ManagementSSL
			} else if node.AlternateAddresses.External.Services.CapiSSL != 0 {
				port = node.AlternateAddresses.External.Services.CapiSSL
			}

			return fmt.Sprintf("https://%s:%d", netutil.ReconstructIPV6(node.AlternateAddresses.External.Hostname),
				port), nil
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
		nodeSummary := values.NodeSummary{
			Version:           node.Version,
			Status:            node.Status,
			NodeUUID:          node.NodeUUID,
			ClusterMembership: node.ClusterMembership,
			Services:          node.Services,
			CPUCount:          node.CPUCount,
			CPUUtil:           node.SystemStats.CPUUtil,
			SwapUsed:          node.SystemStats.SwapUsed,
			SwapTotal:         node.SystemStats.SwapTotal,
			MemFree:           node.SystemStats.MemFree,
			MemTotal:          node.SystemStats.MemTotal,
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
