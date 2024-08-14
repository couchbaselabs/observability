// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package testagent

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/core"
)

// TestHandlers stores the HTTP handlers used by a TestAgent.
// The map key is a [method, endpoint] pair, where the methods are as in http.Method*, and the endpoint always
// has a leading slash.
type TestHandlers map[[2]string]http.HandlerFunc

// NewTestHandler creates the most basic type of handler which will respond with the provided status/body.
func NewTestHandler(t *testing.T, status int, body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)

		_, err := w.Write(body)
		require.NoError(t, err)
	}
}

// NewTestJSONHandler creates a test handler that sends the given payload as JSON.
func NewTestJSONHandler(t *testing.T, status int, payload interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		_, err = w.Write(body)
		require.NoError(t, err)
	}
}

// HealthyHandlers creates a set of TestHandlers for an agent that is running and functioning.
func HealthyHandlers(t *testing.T) TestHandlers {
	return TestHandlers{
		{http.MethodGet, "/agent/api/v1/ping"}: NewTestJSONHandler(t, http.StatusOK, &core.PingResponse{
			State:    core.AgentReady,
			NodeUUID: "N0",
		}),
	}
}
