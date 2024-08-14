// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/couchbase/tools-common/aprov"
	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/cbvalue"
)

type Bootstrapper interface {
	CreateRESTClient() (*Node, error)
}

type nodesSelfData struct {
	UUID string `json:"nodeUUID"`
}

func prepareCluster(rest *cbrest.Client, creds *aprov.Static) (*Node, error) {
	var thisNode *cbrest.Node
	for _, node := range rest.Nodes() {
		if node.BootstrapNode {
			thisNode = node
			break
		}
	}

	var cluster Cluster
	clusterInfoRes, err := rest.Execute(&cbrest.Request{
		Method:             http.MethodGet,
		Endpoint:           cbrest.EndpointPoolsDefault,
		Service:            cbrest.ServiceManagement,
		ExpectedStatusCode: http.StatusOK,
		Idempotent:         true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cluster info: %w", err)
	}
	if err = json.Unmarshal(clusterInfoRes.Body, &cluster); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cluster info: %w", err)
	}

	cluster.UUID = rest.ClusterUUID()

	var node nodesSelfData
	nodeInfoRes, err := rest.Execute(&cbrest.Request{
		Method:             http.MethodGet,
		Endpoint:           "/nodes/self",
		Service:            cbrest.ServiceManagement,
		ExpectedStatusCode: http.StatusOK,
		Idempotent:         true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get self data: %w", err)
	}
	if err = json.Unmarshal(nodeInfoRes.Body, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node info: %w", err)
	}

	version, err := getMinVersion(rest)
	if err != nil {
		return nil, fmt.Errorf("failed to get min version: %w", err)
	}

	nodeStruct := &Node{
		hostname: thisNode.Hostname,
		creds:    creds,
		rest:     rest,
		node:     thisNode,
		uuid:     node.UUID,
		cluster:  cluster,
		version:  version,
	}

	return nodeStruct, nil
}

// getMinVersion returns the miniumm version running in the cluster.
func getMinVersion(client *cbrest.Client) (cbvalue.Version, error) {
	request := &cbrest.Request{
		ContentType:        cbrest.ContentTypeURLEncoded,
		Endpoint:           cbrest.EndpointPoolsDefault,
		ExpectedStatusCode: http.StatusOK,
		Method:             http.MethodGet,
		Service:            cbrest.ServiceManagement,
	}

	response, err := client.Execute(request)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}

	type overlay struct {
		Nodes []struct {
			Version string `json:"version"`
		} `json:"nodes"`
	}

	var decoded *overlay

	err = json.Unmarshal(response.Body, &decoded)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	min := cbvalue.Version(strings.Split(decoded.Nodes[0].Version, "-")[0])

	for _, node := range decoded.Nodes {
		nodeVersion := cbvalue.Version(strings.Split(node.Version, "-")[0])
		if nodeVersion < min {
			min = nodeVersion
		}
	}

	return min, nil
}
