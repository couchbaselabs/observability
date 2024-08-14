// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package bootstrap

import (
	"github.com/couchbase/tools-common/aprov"
	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/cbvalue"
)

type Cluster struct {
	UUID string `json:"-"`
	Name string `json:"clusterName"`
}

type Node struct {
	hostname string
	creds    aprov.Provider
	rest     *cbrest.Client
	node     *cbrest.Node
	cluster  Cluster
	uuid     string
	version  cbvalue.Version
}

func (n *Node) Hostname() string {
	return n.node.Hostname
}

func (n *Node) UUID() string {
	return n.uuid
}

func (n *Node) Cluster() Cluster {
	return n.cluster
}

func (n *Node) Close() error {
	n.rest.Close()
	return nil
}

func (n *Node) RestClient() *cbrest.Client {
	return n.rest
}

func (n *Node) Credentials() (string, string) {
	return n.creds.GetCredentials("")
}

func (n *Node) GetServicePort(service cbrest.Service) (int, error) {
	return int(n.node.GetPort(service, n.rest.TLS(), n.rest.AltAddr())), nil
}

func (n *Node) HasService(service cbrest.Service) (bool, error) {
	return n.node.GetPort(service, n.rest.TLS(), n.rest.AltAddr()) > 0, nil
}

func (n *Node) Services() *cbrest.Services {
	return n.node.Services
}

func (n *Node) ClusterUUID() string {
	return n.cluster.UUID
}

// Version returns the minimum version running in the cluster.
func (n *Node) Version() cbvalue.Version {
	return n.version
}
