// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package parsers

import (
	"regexp"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"
)

var (
	// rebalanceStartRegexp captures the kept and ejected nodes along with the operation id
	rebalanceStartRegexp = regexp.MustCompile(`KeepNodes\s=\s\[(?P<nodesIn>[^\]]*)\],\sEjectNodes\s=\s\[` +
		`(?P<nodesOut>[^\]]*)\].*Operation\sId\s=\s(?P<operation>.*)`)
	// rebalanceEndRegexp captures the reason the rebalance succeeded/failed and the operation id
	rebalanceEndRegexp = regexp.MustCompile(`(?P<reason>Rebalance.*)\..*Rebalance\sOperation\sId\s=\s(?P<operation>.*)`)
	// failoverStartRegexp captures the nodes being failed over along with the operation id
	failoverStartRegexp = regexp.MustCompile(`Starting\sfailover\sof\snodes\s\['(?P<node>[^']*)'\].*` +
		`Operation\sId\s=\s(?P<operation>.*)`)
	// failoverEndRegexp captures the reason the failover succeeded/failed and the operation id
	failoverEndRegexp = regexp.MustCompile(`(?P<reason>(?:Fail|Graceful|Hard).*)\..*Operation\sId\s=\s` +
		`(?P<operation>.*)`)
	// nodeJoinedRegexp captures which node has joined the cluster
	nodeJoinedRegexp = regexp.MustCompile(`Node\s(?P<node>[^\s]*)\sjoined\scluster`)
	// wentDownRegexp captures which node has gone down
	wentDownRegexp = regexp.MustCompile(`saw\sthat\snode\s'(?P<node>[^']+)'\swent\sdown.*nodedown_reason,` +
		`(?P<reason>[^}]*)`)
)

// RebalanceStartTime gets the times rebalances start at.
// Example line: 2021-05-28T16:26:30.435+01:00, ns_orchestrator:0:info:message(n_0@cb.local) - Starting rebalance,
//
//	KeepNodes = ['n_0@cb.local'], EjectNodes = [], Failed over and being ejected nodes = []; no delta recovery nodes;
//	Operation Id = f397378beb69a75d434932f1cbbf554c
func RebalanceStartTime(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"Starting rebalance"}, nil, rebalanceStartRegexp, 4)
	if err == values.ErrNotInLine {
		return nil, err
	}

	if !strings.Contains(line, "Operation Id") {
		return nil, values.ErrNotFullLine
	}

	if err != nil {
		return nil, err
	}

	nodesIn := formatNodeList(output[1])

	nodesOut := formatNodeList(output[2])

	return &values.Result{
		Event:       values.RebalanceStartEvent,
		NodesIn:     nodesIn,
		NodesOut:    nodesOut,
		OperationID: output[3],
	}, nil
}

// RebalanceFinish gets when a rebalance finishes and if it was successful.
// Example successful line: 2021-05-28T16:26:32.989+01:00, ns_orchestrator:0:info:message(n_0@cb.local) - Rebalance
//
//	completed successfully.Rebalance Operation Id = f397378beb69a75d434932f1cbbf554c
//
// Example failed line: 2021-05-28T02:30:52.547-07:00, ns_orchestrator:0:critical:message(n_0@cb.local) - Rebalance
//
//	exited with reason {pre_rebalance_janitor_run_failed,"MLE_Event_Meta",{error,wait_for_
//	memcached_failed,['n_0@cb.local']}}.Rebalance Operation Id = cfa8d93e5e8996804cf13329b29ae6ce
func RebalanceFinish(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, nil, []string{"Rebalance completed", "Rebalance exited"}, rebalanceEndRegexp, 3)
	if err == values.ErrNotInLine {
		return nil, err
	}

	if !strings.Contains(line, "Operation Id") {
		return nil, values.ErrNotFullLine
	}

	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:       values.RebalanceFinishEvent,
		Successful:  strings.Contains(output[1], "successfully"),
		Reason:      output[1],
		OperationID: output[2],
	}, nil
}

// FailoverStartTime gets the times failovers start at.
// Example line: 2021-05-18T14:26:37.758-07:00, ns_orchestrator:0:info:message(n_0@cb.local) -
//
//	Starting failover of nodes ['n_0@cb.local']. Operation Id = e12ad821847196a12b0fd7885e1b539f
func FailoverStartTime(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"Starting failover"}, nil, failoverStartRegexp, 3)
	if err == values.ErrNotInLine {
		return nil, err
	}

	if !strings.Contains(line, "Operation Id") {
		return nil, values.ErrNotFullLine
	}

	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:       values.FailoverStartEvent,
		Node:        output[1],
		OperationID: output[2],
	}, nil
}

// FailoverFinish gets when a failover finishes and if it was successful.
// Example successful line: 2021-06-14T13:48:27.324+01:00, ns_orchestrator:0:info:message(n_0@cb.local) - Failover
//
//	completed successfully.Rebalance Operation Id = 40d3ad52949ad0d86147851c0129b54a
//
// Example failed line: 2021-06-15T18:27:30.134-07:00, ns_orchestrator:0:critical:message(n_0@cb.local) - Failover
//
//	exited with reason {{timeout,{gen_server,call,[ns_config_rep,synchronize,30000]}},{gen_server,call,[<0.8318.0>,
//	{set_quorum_nodes,{set,11,16,16,8,80,48,{{[],['n_1@cb.local','n_2@cb.local']}}}},infinity]}}.Rebalance Operation
//	Id = e7750adef91174c55874e3578b39ddae
func FailoverFinish(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"ailover"}, []string{"completed", "exited"}, failoverEndRegexp, 3)
	if err == values.ErrNotInLine {
		return nil, err
	}

	if !strings.Contains(line, "Operation Id") {
		return nil, values.ErrNotFullLine
	}

	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:       values.FailoverEndEvent,
		Successful:  strings.Contains(output[1], "successfully"),
		Reason:      output[1],
		OperationID: output[2],
	}, nil
}

// NodeJoinedCluster gets when a node joins the cluster.
// Example line: 2021-02-22T09:30:45.540Z, ns_cluster:3:info:message(ns_1@10.144.210.102) - Node ns_1@10.144.210.102
//
//	joined cluster
func NodeJoinedCluster(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"joined cluster"}, nil, nodeJoinedRegexp, 2)
	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event: values.NodeJoinedEvent,
		Node:  output[1],
	}, nil
}

// NodeWentDown gets which nodes went down and why.
// Example line: 2021-05-18T14:27:18.619-07:00, ns_node_disco:5:warning:node down(ns_1@lgpecpdb0000988.gso.aexp.com) -
//
//	Node 'ns_1@lgpecpdb0000988.gso.aexp.com' saw that node 'ns_1@lgpecpdb0001034.gso.aexp.com' went down. Details:
//	[{nodedown_reason,connection_closed}]
func NodeWentDown(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"went down", "saw that"}, nil, wentDownRegexp, 3)
	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:  values.NodeWentDownEvent,
		Node:   output[1],
		Reason: output[2],
	}, nil
}

// formatNodeList converts from format: 'ns_1@10.144.210.101','ns_1@10.144.210.102'
// to format: []string{"ns_1@10.144.210.101", "ns_1@10.144.210.102"}.
func formatNodeList(nodeString string) []string {
	nodeList := strings.Split(nodeString, ",")
	// if empty return empty slice
	if len(nodeList) == 1 && nodeList[0] == "" {
		return []string{}
	}

	for i, node := range nodeList {
		// remove '' from around node
		nodeList[i] = node[1 : len(node)-1]
	}

	return nodeList
}
