package parsers

import (
	"regexp"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/log/values"
)

// RebalanceStartTime gets the times rebalances start at.
// Example line: [user:info,2021-02-22T09:30:55.511Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:idle:774]Starting
//	rebalance, KeepNodes = ['ns_1@10.144.210.101','ns_1@10.144.210.102'], EjectNodes = [], Failed over and
//	being ejected nodes = []; no delta recovery nodes; Operation Id = 4e5e8183c472a43032584fd88be0e440
func RebalanceStartTime(line string) (*values.Result, error) {
	if !strings.Contains(line, "Starting rebalance") {
		return nil, values.ErrNotInLine
	}

	if !strings.Contains(line, "Operation Id") {
		return nil, values.ErrNotFullLine
	}

	lineRegexp := regexp.MustCompile(`KeepNodes\s=\s\[(?P<nodesIn>[^\]]*)\],\sEjectNodes\s=\s\[(?P<nodesOut>[^\]]*)` +
		`\].*Operation\sId\s=\s(?P<operation>.*)`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)

	if len(output) == 0 || len(output[0]) < 4 {
		return nil, values.ErrRegexpMissingFields
	}

	nodesIn := formatNodeList(output[0][1])

	nodesOut := formatNodeList(output[0][2])

	return &values.Result{
		Event:       values.RebalanceStartEvent,
		NodesIn:     nodesIn,
		NodesOut:    nodesOut,
		OperationID: output[0][3],
	}, nil
}

// RebalanceFinish gets when a rebalance finishes and if it was successful.
// Example successful line: [user:info,2021-02-25T14:51:11.951Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_
//	rebalance_completion:1402]Rebalance completed successfully.Rebalance Operation Id = f89c0620759d3d341bf3bc0a8e5f03ec
// Example failed line: [user:info,2021-03-04T14:14:27.445Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_
//	rebalance_completion:1402]Rebalance stopped by user.Rebalance Operation Id = d40716c88f2dfc54640df31ba51b8b71
func RebalanceFinish(line string) (*values.Result, error) {
	if !strings.Contains(line, "]Rebalance") || !strings.Contains(line, "log_rebalance_completion") {
		return nil, values.ErrNotInLine
	}

	if !strings.Contains(line, "Operation Id") {
		return nil, values.ErrNotFullLine
	}

	lineRegexp := regexp.MustCompile(`(?P<reason>Rebalance[^\.]*)\..*Operation\sId\s=\s(?P<operation>.*)`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)

	if len(output) == 0 || len(output[0]) < 3 {
		return nil, values.ErrRegexpMissingFields
	}

	return &values.Result{
		Event:       values.RebalanceFinishEvent,
		Successful:  strings.Contains(output[0][1], "successfully"),
		Reason:      output[0][1],
		OperationID: output[0][2],
	}, nil
}

// FailoverStartTime gets the times failovers start at.
// Example line: [user:info,2021-02-25T14:50:34.396Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:handle_start_
//	failover:1571]Starting failover of nodes ['ns_1@10.144.210.102']. Operation Id = 8d06a862a8bf0d0a47c083f352b05d29
func FailoverStartTime(line string) (*values.Result, error) {
	if !strings.Contains(line, "Starting failover") {
		return nil, values.ErrNotInLine
	}

	if !strings.Contains(line, "Operation Id") {
		return nil, values.ErrNotFullLine
	}

	lineRegexp := regexp.MustCompile(`Starting\sfailover\sof\snodes\s\['(?P<node>[^']*)'\].*Operation\sId\s=\s` +
		`(?P<operation>.*)`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)

	if len(output) == 0 || len(output[0]) < 3 {
		return nil, values.ErrRegexpMissingFields
	}

	return &values.Result{
		Event:       values.FailoverStartEvent,
		Node:        output[0][1],
		OperationID: output[0][2],
	}, nil
}

// FailoverFinish gets when a failover finishes and if it was successful.
// Example successful line: [user:info,2021-02-25T14:50:36.330Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_
//	rebalance_completion:1402]Failover completed successfully.Rebalance Operation Id = 8d06a862a8bf0d0a47c083f352b05d29
// Example failed line: [user:error,2021-03-05T10:27:51.937Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_
//	rebalance_completion:1402]Graceful failover exited with reason {{badmatch,{leader_activities_error,{default,graceful
//	_failover},{quorum_lost,{lease_lost,'ns_1@10.144.210.102'}}}},[{ns_rebalancer,run_graceful_failover,1,[
//	{file,"src/ns_rebalancer.erl"},{line,1178}]},{proc_lib,init_p_do_apply,3,[{file,"proc_lib.erl"},{line,249}]}
//	]}.Rebalance Operation Id = e7750adef91174c55874e3578b39ddae
func FailoverFinish(line string) (*values.Result, error) {
	if !strings.Contains(line, "log_rebalance_completion") || !strings.Contains(line, "ailover") {
		return nil, values.ErrNotInLine
	}

	if !strings.Contains(line, "Operation Id") {
		return nil, values.ErrNotFullLine
	}

	lineRegexp := regexp.MustCompile(`(?P<reason>(?:Fail|Graceful|Hard).*)\.Rebalance\sOperation\sId\s=\s` +
		`(?P<operation>.*)`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)

	if len(output) == 0 || len(output[0]) < 3 {
		return nil, values.ErrRegexpMissingFields
	}

	return &values.Result{
		Event:       values.FailoverEndEvent,
		Successful:  strings.Contains(output[0][1], "successfully"),
		Reason:      output[0][1],
		OperationID: output[0][2],
	}, nil
}

// NodeJoinedCluster gets when a node joins the cluster.
// Example line: 2021-02-22T09:30:45.540Z, ns_cluster:3:info:message(ns_1@10.144.210.102) - Node ns_1@10.144.210.102
//	joined cluster
func NodeJoinedCluster(line string) (*values.Result, error) {
	if !strings.Contains(line, "joined cluster") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`Node\s(?P<node>.+)\sjoined\scluster`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 2 {
		return nil, values.ErrRegexpMissingFields
	}

	return &values.Result{
		Event: values.NodeJoinedEvent,
		Node:  output[0][1],
	}, nil
}

// NodeWentDown gets which nodes went down and why.
// Example line: [user:warn,2021-02-19T13:19:36.405Z,nonode@nohost:ns_node_disco<0.361.0>:ns_node_disco:handle_info:189]
//	Node nonode@nohost saw that node 'ns_1@cb.local' went down. Details: [{nodedown_reason,net_kernel_terminated}]
func NodeWentDown(line string) (*values.Result, error) {
	if !strings.Contains(line, "went down") || !strings.Contains(line, "saw that") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`saw\sthat\snode\s'(?P<node>[^']+)'\swent\sdown.*Details:\s(?P<reason>.*)`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 3 {
		return nil, values.ErrRegexpMissingFields
	}

	return &values.Result{
		Event:  values.NodeWentDownEvent,
		Node:   output[0][1],
		Reason: output[0][2],
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
