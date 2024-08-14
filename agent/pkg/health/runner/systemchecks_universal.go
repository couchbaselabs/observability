// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package runner

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/couchbase/tools-common/aprov"
	"github.com/couchbase/tools-common/cbrest"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/bootstrap"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/logger"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/meta"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func getUniversalCheckers() map[string]checkerFn {
	return map[string]checkerFn{
		values.CheckN2NCommunication: checkNodeToNodeCommunication,
	}
}

// NOTE: this won't pick up certain internal ports - CMOS-325 tracks checking *all* ports
var allServices = []cbrest.Service{
	cbrest.ServiceManagement,
	cbrest.ServiceData,
	cbrest.ServiceGSI,
	cbrest.ServiceQuery,
	cbrest.ServiceSearch,
	cbrest.ServiceAnalytics,
	cbrest.ServiceEventing,
	cbrest.ServiceBackup,
	cbrest.ServiceViews,
}

func checkNodeToNodeCommunication(self *bootstrap.Node) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckN2NCommunication,
			Status: values.MissingCheckerStatus,
			Time:   time.Now().UTC(),
		},
		Cluster: self.RestClient().ClusterUUID(),
		Node:    self.UUID(),
	}

	// The agent's REST client is created using ThisNodeOnly, so create a new, temporary one that knows about the other
	// nodes. Disable CCP because this client won't be in use long enough to need it.
	username, password := self.Credentials()
	rest, err := cbrest.NewClient(cbrest.ClientOptions{
		ConnectionString: "couchbase://localhost",
		Provider: &aprov.Static{
			UserAgent: fmt.Sprintf("cbhealthagent/%s", meta.Version),
			Username:  username,
			Password:  password,
		},
		DisableCCP:     true,
		ConnectionMode: cbrest.ConnectionModeDefault,
		Logger:         logger.NewToolsCommonLogger(zap.L().Sugar()),
	})
	if err != nil {
		result.Error = fmt.Errorf("failed to initialise REST client: %w", err)
		return result
	}
	defer rest.Close()

	type problem struct {
		Node    string
		Service cbrest.Service
		Port    uint16
		Error   string
	}
	problems := make([]problem, 0)

	for _, node := range rest.Nodes() {
		if node.BootstrapNode {
			continue
		}
		for _, svc := range allServices {
			// shouldn't need alternate addressing inside the cluster
			port := node.GetPort(svc, rest.TLS(), false)
			if port == 0 {
				continue
			}
			// Just perform a TCP dial and check if the port is open.
			conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", node.GetHostname(false), port))
			if err != nil {
				problems = append(problems, problem{
					Node:    node.Hostname,
					Port:    port,
					Service: svc,
					Error:   err.Error(),
				})
			} else {
				conn.Close()
			}
		}
	}

	if len(problems) == 0 {
		result.Result.Status = values.GoodCheckerStatus
		return result
	}
	result.Result.Status = values.WarnCheckerStatus
	result.Result.Remediation = "Errors were observed when connecting to some of the other nodes in the cluster. " +
		"Please ensure there is no firewall or other network configuration blocking the necessary ports."
	result.Result.Value, _ = json.Marshal(&problems)
	return result
}
