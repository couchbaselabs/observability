// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package agentport

import (
	"net/http"
	"strings"
	"testing"

	"github.com/couchbase/tools-common/aprov"
	"github.com/couchbase/tools-common/restutil"
	"github.com/stretchr/testify/require"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/core"
	"github.com/couchbaselabs/cbmultimanager/agent/testagent"
)

func TestAgentPortReady(t *testing.T) {
	ta := testagent.NewTestAgent(t, true, testagent.HealthyHandlers(t))
	defer ta.Close()

	ap, err := NewAgentPort(ta.Hostname(), int(ta.Port()), &aprov.Static{})
	require.NoError(t, err)
	require.NoError(t, ap.Close())
}

func TestAgentPortRequest(t *testing.T) {
	handlers := testagent.HealthyHandlers(t)
	handlers[[2]string{http.MethodGet, "/test"}] = testagent.NewTestHandler(t, http.StatusOK, []byte(`"ok"`))
	ta := testagent.NewTestAgent(t, true, handlers)
	defer ta.Close()

	ap, err := NewAgentPort(ta.Hostname(), int(ta.Port()), &aprov.Static{})
	require.NoError(t, err)

	var result string

	err = TestRequest(&result).Execute(ap)
	require.NoError(t, err)
	require.Equal(t, "ok", result)

	require.NoError(t, ap.Close())
}

func TestAgentPortInit(t *testing.T) {
	ta := testagent.NewTestAgent(t, false, testagent.HealthyHandlers(t))
	defer ta.Close()

	ap, err := NewAgentPort(ta.Hostname(), int(ta.Port()), &aprov.Static{})
	require.NoError(t, err)
	require.NoError(t, ap.Close())
}

func TestAgentPortInitFailure(t *testing.T) {
	ta := testagent.NewTestAgent(t, false, nil)
	defer ta.Close()
	ta.ReplaceWaitingHandlers(testagent.TestHandlers{
		{http.MethodGet, "/agent/api/v1/ping"}: testagent.NewTestJSONHandler(t, http.StatusOK,
			&core.PingResponse{State: core.AgentWaiting}),
		{http.MethodPost, "/agent/api/v1/activate"}: func(w http.ResponseWriter, r *http.Request) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusInternalServerError,
			}, w, nil)
		},
	})

	ap, err := NewAgentPort(ta.Hostname(), int(ta.Port()), &aprov.Static{})
	require.Error(t, err)
	ap.Close()
}

func TestAgentPortReInit(t *testing.T) {
	handlers := testagent.HealthyHandlers(t)
	handlers[[2]string{http.MethodGet, "/test"}] = testagent.NewTestHandler(t, http.StatusOK, []byte(`"ok"`))
	ta := testagent.NewTestAgent(t, false, handlers)
	defer ta.Close()

	ap, err := NewAgentPort(ta.Hostname(), int(ta.Port()), &aprov.Static{})
	require.NoError(t, err)
	defer ap.Close()

	var result string

	err = TestRequest(&result).Execute(ap)
	require.NoError(t, err)

	// Now, reset the test agent, and do another request
	ta.SetReady(false)

	err = TestRequest(&result).Execute(ap)
	require.NoError(t, err)

	// Finally, check that there have been two requests to /activate.
	activateRequests := 0
	for _, req := range ta.Requests() {
		if strings.HasSuffix(req.URL.Path, "/activate") {
			activateRequests++
		}
	}
	require.Equal(t, 2, activateRequests)
}

func TestAgentPortReInitFailure(t *testing.T) {
	// First, allow the test agent to activate successfully
	handlers := testagent.HealthyHandlers(t)
	handlers[[2]string{http.MethodGet, "/test"}] = testagent.NewTestHandler(t, http.StatusOK, []byte(`"ok"`))
	ta := testagent.NewTestAgent(t, false, handlers)
	defer ta.Close()

	ap, err := NewAgentPort(ta.Hostname(), int(ta.Port()), &aprov.Static{})
	require.NoError(t, err)
	defer ap.Close()

	var result string
	err = TestRequest(&result).Execute(ap)
	require.NoError(t, err)

	// Now, set its state to back waiting, and replace the activate handler with one that will always error
	ta.SetReady(false)
	ta.ReplaceWaitingHandlers(testagent.TestHandlers{
		{http.MethodGet, "/agent/api/v1/ping"}: testagent.NewTestJSONHandler(t, http.StatusOK,
			&core.PingResponse{State: core.AgentWaiting}),
		{http.MethodPost, "/agent/api/v1/activate"}: func(w http.ResponseWriter, r *http.Request) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusInternalServerError,
			}, w, nil)
		},
	})
	err = TestRequest(&result).Execute(ap)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tried to reactivate but failed")
}
