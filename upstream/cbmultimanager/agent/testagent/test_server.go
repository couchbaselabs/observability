// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package testagent

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/couchbase/tools-common/netutil"
	"github.com/couchbase/tools-common/restutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/core"
)

type atomicSlice struct {
	data []interface{}
	mux  sync.RWMutex
}

func (a *atomicSlice) Append(val interface{}) {
	a.mux.Lock()
	defer a.mux.Unlock()

	a.data = append(a.data, val)
}

func (a *atomicSlice) Iter() <-chan interface{} {
	ch := make(chan interface{})

	go func() {
		a.mux.RLock()
		defer a.mux.RUnlock()
		for _, val := range a.data {
			ch <- val
		}
		close(ch)
	}()

	return ch
}

type TestAgent struct {
	t               *testing.T
	server          *httptest.Server
	handlers        atomic.Value
	waitingHandlers atomic.Value
	ready           *atomic.Bool
	requests        atomicSlice
}

func (t *TestAgent) handleRequest(w http.ResponseWriter, r *http.Request) {
	t.requests.Append(*r)

	ready := t.ready.Load()
	reqHandlers := t.handlers.Load().(TestHandlers)
	if !ready {
		reqHandlers = t.waitingHandlers.Load().(TestHandlers)
	}

	testHandler, ok := reqHandlers[[2]string{r.Method, r.URL.Path}]
	if !ok {
		w.Header().Set(core.AgentActiveHeader, fmt.Sprintf("%t", ready))
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	testHandler.ServeHTTP(w, r)
}

func NewTestAgent(t *testing.T, readyWhenStarted bool, handlers TestHandlers) *TestAgent {
	if handlers == nil {
		handlers = make(TestHandlers)
	}
	ta := &TestAgent{
		t:               t,
		ready:           atomic.NewBool(readyWhenStarted),
		handlers:        atomic.Value{},
		waitingHandlers: atomic.Value{},
	}
	ta.handlers.Store(handlers)
	ta.waitingHandlers.Store(TestHandlers{
		{http.MethodGet, "/agent/api/v1/ping"}: NewTestJSONHandler(t, http.StatusOK,
			&core.PingResponse{State: core.AgentWaiting}),
		{http.MethodPost, "/agent/api/v1/activate"}: func(w http.ResponseWriter, r *http.Request) {
			if !ta.ready.CAS(false, true) {
				t.Fatal("Agent was already initialized when it received a call to /activate!")
			}
			restutil.MarshalAndSend(http.StatusOK, "ok", w, nil)
		},
	})

	ta.server = httptest.NewServer(http.HandlerFunc(ta.handleRequest))
	return ta
}

func (t *TestAgent) Close() {
	t.server.Close()
}

// SetReady resets this test agent's ready status to the given value.
func (t *TestAgent) SetReady(ready bool) {
	t.ready.Store(ready)
}

func (t *TestAgent) ReplaceHandlers(handlers TestHandlers) {
	t.handlers.Store(handlers)
}

func (t *TestAgent) ReplaceWaitingHandlers(handlers TestHandlers) {
	t.waitingHandlers.Store(handlers)
}

// URL returns the fully qualified URL which can be used to connect to the cluster.
func (t *TestAgent) URL() string {
	return t.server.URL
}

// Hostname returns the cluster hostname, for the time being this will always be "localhost".
func (t *TestAgent) Hostname() string {
	return "localhost"
}

// Address returns the address of the cluster, for the time being should always be "127.0.0.1".
func (t *TestAgent) Address() string {
	trimmed := netutil.TrimSchema(t.server.URL)
	return trimmed[:strings.Index(trimmed, ":")]
}

// Port returns the port which requests should be sent to; this will be the same port for all services.
//
// NOTE: This port is randomly selected at runtime and will therefore vary.
func (t *TestAgent) Port() uint16 {
	testURL, err := url.Parse(t.server.URL)
	require.NoError(t.t, err)

	parsed, err := strconv.Atoi(testURL.Port())
	require.NoError(t.t, err)

	return uint16(parsed)
}

// Requests returns a slice of all requests seen by this TestAgent.
func (t *TestAgent) Requests() []http.Request {
	result := make([]http.Request, 0)
	for req := range t.requests.Iter() {
		result = append(result, req.(http.Request))
	}
	return result
}
