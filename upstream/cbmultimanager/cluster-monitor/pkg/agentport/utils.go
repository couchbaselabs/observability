// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package agentport

import (
	"encoding/json"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/core"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func PingAgent(status *core.PingResponse) *Request {
	return NewRequest((*AgentPort).Get, "/agent/api/v1/ping", nil, status)
}

func ActivateAgent(payload *core.ActivateRequest, result *string) *Request {
	data, err := json.Marshal(payload)
	if err != nil {
		return NewRequestError(err)
	}
	return NewRequest((*AgentPort).Post, "/agent/api/v1/activate", data, result)
}

func CheckerResults(results *map[string]values.WrappedCheckerResult) *Request {
	return NewRequest((*AgentPort).Get, "/api/v1/checkers", nil, results).WithRevive()
}

func TestRequest(result *string) *Request {
	return NewRequest((*AgentPort).Get, "/test", nil, result).WithRevive()
}
