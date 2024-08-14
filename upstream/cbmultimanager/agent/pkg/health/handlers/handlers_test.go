// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/store"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func TestGetCheckers(t *testing.T) {
	r := mux.NewRouter()
	memStore := store.NewInMemoryStore()
	RegisterHealthAgentRoutes(r, memStore)
	server := httptest.NewServer(r)
	defer server.Close()

	res, err := http.Get(fmt.Sprintf("%s/api/v1/checkers", server.URL))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	defer res.Body.Close()

	var checkers map[string]*values.WrappedCheckerResult
	require.NoError(t, json.NewDecoder(res.Body).Decode(&checkers))
	require.Len(t, checkers, 0)

	memStore.SetCheckerResult("c-1", &values.WrappedCheckerResult{Result: &values.CheckerResult{Name: "c-1"}})
	memStore.SetCheckerResult("c-2", &values.WrappedCheckerResult{Result: &values.CheckerResult{Name: "c-2"}})

	res, err = http.Get(fmt.Sprintf("%s/api/v1/checkers", server.URL))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	defer res.Body.Close()

	require.NoError(t, json.NewDecoder(res.Body).Decode(&checkers))
	require.Len(t, checkers, 2)
	require.Equal(t, map[string]*values.WrappedCheckerResult{
		"c-1": {Result: &values.CheckerResult{Name: "c-1"}},
		"c-2": {Result: &values.CheckerResult{Name: "c-2"}},
	}, checkers)
}

func TestGetCheckerByName(t *testing.T) {
	r := mux.NewRouter()
	memStore := store.NewInMemoryStore()
	RegisterHealthAgentRoutes(r, memStore)
	server := httptest.NewServer(r)
	defer server.Close()

	memStore.SetCheckerResult("c-1", &values.WrappedCheckerResult{Result: &values.CheckerResult{Name: "c-1"}})

	res, err := http.Get(fmt.Sprintf("%s/api/v1/checkers/abc", server.URL))
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	require.NoError(t, res.Body.Close())

	res, err = http.Get(fmt.Sprintf("%s/api/v1/checkers/c-1", server.URL))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	var checker *values.WrappedCheckerResult
	require.NoError(t, json.NewDecoder(res.Body).Decode(&checker))
	require.NoError(t, res.Body.Close())
	require.Equal(t, &values.WrappedCheckerResult{Result: &values.CheckerResult{Name: "c-1"}}, checker)
}
