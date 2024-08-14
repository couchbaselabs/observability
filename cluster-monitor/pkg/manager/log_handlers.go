// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/restutil"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func (m *Manager) getLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid, ok := m.convertAliasToUUID(vars["uuid"], w)
	if !ok {
		return
	}

	logName := vars["logName"]

	nodeUUID, err := url.PathUnescape(vars["nodeUUID"])
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "invalid node uuid",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	cluster, err := m.store.GetCluster(uuid, true)
	if err != nil {
		if errors.Is(err, values.ErrNotFound) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusNotFound,
				Msg:    "cluster with given uuid does not exist",
			}, w, nil)
			return
		}

		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get cluster information",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	if !cluster.Enterprise {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "Can only get logs for Enterprise Edition clusters",
		}, w, nil)
		return
	}

	var nodeHost string
	for _, node := range cluster.NodesSummary {
		if node.NodeUUID == nodeUUID {
			nodeHost = node.Host
			break
		}
	}

	if nodeHost == "" {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusNotFound,
			Msg:    fmt.Sprintf("node with uuid '%s' not found", nodeUUID),
		}, w, nil)
		return
	}

	client, err := couchbase.NewClient([]string{nodeHost}, cluster.User, cluster.Password, cluster.GetTLSConfig(), false)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not connect to remote cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	res, err := client.GetSASLLogs(ctx, logName)
	if err != nil {
		if errors.Is(err, values.ErrNotFound) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusNotFound,
				Msg:    "resource does not exist",
			}, w, nil)
			return
		}

		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get log files",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	defer res.Close()

	_, err = io.Copy(w, res)
	if err != nil {
		zap.S().Errorw("(HTTP Manager) Could not copy logs", "err", err)
	}
}
