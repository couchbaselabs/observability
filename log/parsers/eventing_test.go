package parsers

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/log/values"
)

func TestEventFunctionDeployedOrUndeployed(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLineEventFunctionDeployed",
			Line: `2021-03-29T09:06:11.364+00:00 [Info] ServiceMgr::setSettings Function: eventing_func settings params` +
				`: map[deployment_status:true feed-boundary:from-now processing_status:true]`,
			ExpectedResult: &values.Result{
				Event:    values.EventingFunctionDeployedEvent,
				Function: "eventing_func",
			},
		},
		{
			Name: "inLineEventingFunctionUndeployed",
			Line: `2021-03-29T09:06:31.061+00:00 [Info] ServiceMgr::setSettings Function: eventing_func settings params` +
				`: map[deployment_status:false processing_status:false]`,
			ExpectedResult: &values.Result{
				Event:    values.EventingFunctionUndeployedEvent,
				Function: "eventing_func",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-03-29T09:06:31.061+00:00 [Info] ServiceMgr::setSettings Function: eventing_func settings` +
				` map[deployment_status:false processing_status:false]`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, EventFunctionDeployedOrUndeployed)
}
