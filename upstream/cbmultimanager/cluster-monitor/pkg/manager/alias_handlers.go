// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"net/http"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/restutil"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func (m *Manager) AddAlias(w http.ResponseWriter, r *http.Request) {
	alias := mux.Vars(r)["alias"]

	var body struct {
		ClusterUUID string `json:"cluster_uuid"`
	}

	if !restutil.DecodeJSONRequestBody(r.Body, &body, w) {
		return
	}

	if len(alias) == 0 || len(alias) > 100 {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "Alias length must be between 1 and 100 characters",
		}, w, nil)
		return
	}

	if !strings.HasPrefix(alias, aliasPrefix) {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "Aliases must start with " + aliasPrefix,
		}, w, nil)
		return
	}

	if body.ClusterUUID == "" {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "cluster_uuid is required",
		}, w, nil)
		return
	}

	if err := m.store.AddAlias(&values.ClusterAlias{Alias: alias, ClusterUUID: body.ClusterUUID}); err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "Could not add alias",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	zap.S().Infow("(Manager) Added alias", "cluster", body.ClusterUUID, "alias", alias)
	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}

func (m *Manager) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	alias := mux.Vars(r)["alias"]

	if err := m.store.DeleteAlias(alias); err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not delete alias",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	zap.S().Infow("(Manager) Deleted alias", "alias", alias)
}
