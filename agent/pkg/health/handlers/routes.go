// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package handlers

import (
	"github.com/gorilla/mux"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/store"
)

func RegisterHealthAgentRoutes(r *mux.Router, store *store.InMemory) {
	v1 := r.PathPrefix("/api/v1").Subrouter()

	handler := HealthHandlers{store: store}

	// The checkers returns the results for all the agent checkers.
	v1.HandleFunc("/checkers", handler.getCheckers).Methods("GET")
	// The endpoint below returns the result for only the checker with the given name.
	v1.HandleFunc("/checkers/{name}", handler.getCheckerByName).Methods("GET")
}
