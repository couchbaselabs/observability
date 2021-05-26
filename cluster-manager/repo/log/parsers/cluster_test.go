package parsers

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/log/values"
)

func TestRebalanceStartTime(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `[user:info,2021-02-22T09:30:55.511Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:idle:774]Starting ` +
				`rebalance, KeepNodes = ['ns_1@10.144.210.101','ns_1@10.144.210.102'], EjectNodes = [], Failed over and ` +
				`being ejected nodes = []; no delta recovery nodes; Operation Id = 4e5e8183c472a43032584fd88be0e440`,
			ExpectedResult: &values.Result{
				Event:       values.RebalanceStartEvent,
				NodesIn:     []string{"ns_1@10.144.210.101", "ns_1@10.144.210.102"},
				NodesOut:    []string{},
				OperationID: "4e5e8183c472a43032584fd88be0e440",
			},
		},
		{
			Name: "notInLine",
			Line: `[user:info,2021-02-22T09:30:55.511Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:idle:774]Start` +
				`ing, KeepNodes = ['ns_1@10.144.210.101','ns_1@10.144.210.102'], EjectNodes = [], Failed over and being ej` +
				`ected nodes = []; no delta recovery nodes; Operation Id = 4e5e8183c472a43032584fd88be0e440`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, RebalanceStartTime)
}

func TestRebalanceFinish(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "fullLineSuccessful",
			Line: `[user:info,2021-02-25T14:51:11.951Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_rebalance_` +
				`completion:1402]Rebalance completed successfully.Rebalance Operation Id = f89c0620759d3d341bf3bc0a8e5f03ec`,
			ExpectedResult: &values.Result{
				Event:       values.RebalanceFinishEvent,
				Successful:  true,
				Reason:      "Rebalance completed successfully",
				OperationID: "f89c0620759d3d341bf3bc0a8e5f03ec",
			},
		},
		{
			Name: "fullLineFailed",
			Line: `[user:info,2021-03-04T14:14:27.445Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_rebalance` +
				`_completion:1402]Rebalance stopped by user.Rebalance Operation Id = d40716c88f2dfc54640df31ba51b8b71`,
			ExpectedResult: &values.Result{
				Event:       values.RebalanceFinishEvent,
				Successful:  false,
				Reason:      "Rebalance stopped by user",
				OperationID: "d40716c88f2dfc54640df31ba51b8b71",
			},
		},
		{
			Name: "partialLine",
			Line: `[user:info,2021-02-25T14:51:11.951Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_rebalance_` +
				`completion:1402]Rebalance completed successfully.`,
			ExpectedError: values.ErrNotFullLine,
		},
		{
			Name: "notInLine",
			Line: `[user:info,2021-02-25T14:51:11.951Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_rebalance_` +
				`completion:1402]Reba completed successfully.`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, RebalanceFinish)
}

func TestFailoverStartTime(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `[user:info,2021-02-25T14:50:34.396Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:handle_start_` +
				`failover:1571]Starting failover of nodes ['ns_1@10.144.210.102']. Operation Id = ` +
				`8d06a862a8bf0d0a47c083f352b05d29`,
			ExpectedResult: &values.Result{
				Event:       values.FailoverStartEvent,
				Node:        "ns_1@10.144.210.102",
				OperationID: "8d06a862a8bf0d0a47c083f352b05d29",
			},
		},
		{
			Name: "notInLine",
			Line: `[user:info,2021-02-25T14:50:34.396Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:handle_start_` +
				`failover:1571]Starting of nodes ['ns_1@10.144.210.102']. Operation Id = ` + `8d06a862a8bf0d0a47c083f352b05d29`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, FailoverStartTime)
}

func TestFailoverFinish(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "fullLineSuccessful",
			Line: `[user:info,2021-02-25T14:50:36.330Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_rebalance_` +
				`completion:1402]Failover completed successfully.Rebalance Operation Id = 8d06a862a8bf0d0a47c083f352b05d29`,
			ExpectedResult: &values.Result{
				Event:       values.FailoverEndEvent,
				Successful:  true,
				Reason:      "Failover completed successfully",
				OperationID: "8d06a862a8bf0d0a47c083f352b05d29",
			},
		},
		{
			Name: "fullLineFailed",
			Line: `[user:error,2021-03-05T10:27:51.937Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_rebalance` +
				`_completion:1402]Graceful failover exited with reason {{badmatch,{leader_activities_error,{default,graceful` +
				`_failover},{quorum_lost,{lease_lost,'ns_1@10.144.210.102'}}}},[{ns_rebalancer,run_graceful_failover,1,[` +
				`{file,"src/ns_rebalancer.erl"},{line,1178}]},{proc_lib,init_p_do_apply,3,[{file,"proc_lib.erl"},{line,249}]}` +
				`]}.Rebalance Operation Id = e7750adef91174c55874e3578b39ddae`,
			ExpectedResult: &values.Result{
				Event:      values.FailoverEndEvent,
				Successful: false,
				Reason: `Graceful failover exited with reason {{badmatch,{leader_activities_error,{default,graceful` +
					`_failover},{quorum_lost,{lease_lost,'ns_1@10.144.210.102'}}}},[{ns_rebalancer,run_graceful_failover,1,[` +
					`{file,"src/ns_rebalancer.erl"},{line,1178}]},{proc_lib,init_p_do_apply,3,[{file,"proc_lib.erl"},{line,249}]}` +
					`]}`,
				OperationID: "e7750adef91174c55874e3578b39ddae",
			},
		},
		{
			Name: "partialLine",
			Line: `[user:info,2021-02-25T14:50:36.330Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_rebalance_` +
				`completion:1402]Failover completed successfully.`,
			ExpectedError: values.ErrNotFullLine,
		},
		{
			Name: "notInLine",
			Line: `[user:info,2021-02-25T14:51:11.951Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_rebalance_` +
				`completion:1402]Reba completed successfully.`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, FailoverFinish)
}

func TestNodeJoinedCluster(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `2021-02-22T09:30:45.540Z, ns_cluster:3:info:message(ns_1@10.144.210.102) - Node ns_1@10.144.210.102 ` +
				`joined cluster`,
			ExpectedResult: &values.Result{
				Event: values.NodeJoinedEvent,
				Node:  "ns_1@10.144.210.102",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-02-22T09:30:45.540Z, ns_cluster:3:info:message(ns_1@10.144.210.102) - Node ns_1@10.144.210.102 ` +
				`joined`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, NodeJoinedCluster)
}

func TestNodeWentDown(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `[user:warn,2021-02-19T13:19:36.405Z,nonode@nohost:ns_node_disco<0.361.0>:ns_node_disco:handle_info:189]` +
				`Node nonode@nohost saw that node 'ns_1@cb.local' went down. Details: [{nodedown_reason,net_kernel_` +
				`terminated}]`,
			ExpectedResult: &values.Result{
				Event:  values.NodeWentDownEvent,
				Node:   "ns_1@cb.local",
				Reason: "[{nodedown_reason,net_kernel_terminated}]",
			},
		},
		{
			Name: "notInLine",
			Line: `[user:warn,2021-02-19T13:19:36.405Z,nonode@nohost:ns_node_disco<0.361.0>:ns_node_disco:handle_info:189]` +
				`Node nonode@nohost saw node 'ns_1@cb.local' go down. Details: [{nodedown_reason,net_kernel_` +
				`terminated}]`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, NodeWentDown)
}
