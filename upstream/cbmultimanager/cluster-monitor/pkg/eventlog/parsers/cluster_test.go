// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package parsers

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"
)

func TestRebalanceStartTime(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `2021-05-28T16:26:30.435+01:00, ns_orchestrator:0:info:message(n_0@cb.local) - Starting rebalance, ` +
				`KeepNodes = ['n_0@cb.local'], EjectNodes = [], Failed over and being ejected nodes = []; no delta recovery ` +
				`nodes; Operation Id = f397378beb69a75d434932f1cbbf554c`,
			ExpectedResult: &values.Result{
				Event:       values.RebalanceStartEvent,
				NodesIn:     []string{"n_0@cb.local"},
				NodesOut:    []string{},
				OperationID: "f397378beb69a75d434932f1cbbf554c",
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
			Line: `2021-05-28T16:26:32.989+01:00, ns_orchestrator:0:info:message(n_0@cb.local) - Rebalance completed ` +
				`successfully.Rebalance Operation Id = f397378beb69a75d434932f1cbbf554c`,
			ExpectedResult: &values.Result{
				Event:       values.RebalanceFinishEvent,
				Successful:  true,
				Reason:      "Rebalance completed successfully",
				OperationID: "f397378beb69a75d434932f1cbbf554c",
			},
		},
		{
			Name: "fullLineFailed",
			Line: `2021-05-28T02:30:52.547-07:00, ns_orchestrator:0:critical:message(ns_1@lgpecpdb0000888.gso.aexp.com) - ` +
				`Rebalance exited with reason {pre_rebalance_janitor_run_failed,"MLE_Event_Meta",{error,wait_for_memcached_` +
				`failed,['ns_1@lgpecpdb0000982.gso.aexp.com']}}.Rebalance Operation Id = cfa8d93e5e8996804cf13329b29ae6ce`,
			ExpectedResult: &values.Result{
				Event:      values.RebalanceFinishEvent,
				Successful: false,
				Reason: `Rebalance exited with reason {pre_rebalance_janitor_run_failed,"MLE_Event_Meta",{error,wait_for` +
					`_memcached_failed,['ns_1@lgpecpdb0000982.gso.aexp.com']}}`,
				OperationID: "cfa8d93e5e8996804cf13329b29ae6ce",
			},
		},
		{
			Name: "partialLine",
			Line: `2021-05-28T16:26:32.989+01:00, ns_orchestrator:0:info:message(n_0@cb.local) - Rebalance completed ` +
				`successfully.`,
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
			Line: `2021-05-18T14:26:37.758-07:00, ns_orchestrator:0:info:message(ns_1@lgpecpdb0000888.gso.aexp.com) - ` +
				`Starting failover of nodes ['ns_1@lgpecpdb0001034.gso.aexp.com']. Operation Id = ` +
				`e12ad821847196a12b0fd7885e1b539f`,
			ExpectedResult: &values.Result{
				Event:       values.FailoverStartEvent,
				Node:        "ns_1@lgpecpdb0001034.gso.aexp.com",
				OperationID: "e12ad821847196a12b0fd7885e1b539f",
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
			Line: `2021-06-14T13:48:27.324+01:00, ns_orchestrator:0:info:message(n_0@192.168.1.139) - Failover completed ` +
				`successfully.Rebalance Operation Id = 40d3ad52949ad0d86147851c0129b54a`,
			ExpectedResult: &values.Result{
				Event:       values.FailoverEndEvent,
				Successful:  true,
				Reason:      "Failover completed successfully",
				OperationID: "40d3ad52949ad0d86147851c0129b54a",
			},
		},
		{
			Name: "fullLineFailed",
			Line: `2021-06-15T18:27:30.134-07:00, ns_orchestrator:0:critical:message(n_0@cb.local) - Failover ` +
				`exited with reason {{timeout,{gen_server,call,[ns_config_rep,synchronize,30000]}},{gen_server,call,[` +
				`<0.8318.0>,{set_quorum_nodes,{set,11,16,16,8,80,48,{{[],['n_1@cb.local','n_2@cb.local']}}}},infinity]}}.` +
				`Rebalance Operation Id = e7750adef91174c55874e3578b39ddae`,
			ExpectedResult: &values.Result{
				Event:      values.FailoverEndEvent,
				Successful: false,
				Reason: `Failover exited with reason {{timeout,{gen_server,call,[ns_config_rep,synchronize,30000]}},` +
					`{gen_server,call,[<0.8318.0>,{set_quorum_nodes,{set,11,16,16,8,80,48,{{[],['n_1@cb.local','n_2@cb.local'` +
					`]}}}},infinity]}}`,
				OperationID: "e7750adef91174c55874e3578b39ddae",
			},
		},
		{
			Name: "partialLine",
			Line: `2021-06-14T13:48:27.324+01:00, ns_orchestrator:0:info:message(n_0@192.168.1.139) - Failover completed ` +
				`successfully.`,
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
			Line: `2021-05-18T14:27:18.619-07:00, ns_node_disco:5:warning:node down(ns_1@cb.local) -  ` +
				`Node 'ns_0@cb.local' saw that node 'ns_1@cb.local' went down. ` +
				`Details: [{nodedown_reason,connection_closed}]`,
			ExpectedResult: &values.Result{
				Event:  values.NodeWentDownEvent,
				Node:   "ns_1@cb.local",
				Reason: "connection_closed",
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
