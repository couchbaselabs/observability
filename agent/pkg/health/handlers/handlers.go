// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package handlers

import (
	"net/http"

	"github.com/couchbase/tools-common/restutil"
	"github.com/gorilla/mux"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/store"
)

type HealthHandlers struct {
	store *store.InMemory
}

func (h *HealthHandlers) getCheckers(w http.ResponseWriter, _ *http.Request) {
	restutil.MarshalAndSend(http.StatusOK, h.store.GetCheckers(), w, nil)
}

func (h *HealthHandlers) getCheckerByName(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	checker, err := h.store.GetCheckerResult(name)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusNotFound,
			Msg:    "checker with that name does not exist or has not been run yet",
		}, w, nil)
		return
	}

	restutil.MarshalAndSend(http.StatusOK, checker, w, nil)
}
