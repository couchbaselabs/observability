// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/restutil"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// dismissalRequest is a wrapper around dismissal. It will be used so that the user can provide a dismiss_for that will
// be translated by the system to the dismissal "Until" field.
type dismissalRequest struct {
	values.Dismissal
	DismissFor string `json:"dismiss_for"`
}

func (m *Manager) dismiss(w http.ResponseWriter, r *http.Request) {
	var dismissal dismissalRequest
	if !restutil.DecodeJSONRequestBody(r.Body, &dismissal, w) {
		return
	}

	// validate the dismissal
	// 1st check that the time constraint is valid, either it has to be forever or (exclusive) a dismiss_for has to be
	// provided
	if (!dismissal.Forever && dismissal.DismissFor == "") || (dismissal.Forever && dismissal.DismissFor != "") {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "either 'forever' or 'dismiss_for' have to be provided",
		}, w, nil)
		return
	}

	dismissal.Until = time.Time{}

	// if dismiss for provided make sure is a valid duration string
	if dismissal.DismissFor != "" {
		duration, err := time.ParseDuration(dismissal.DismissFor)
		if err != nil {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusBadRequest,
				Msg:    "invalid duration string for dismiss_for",
				Extras: err.Error(),
			}, w, nil)
			return
		}

		dismissal.Until = time.Now().Add(duration)
	}

	// validate that a checker name was given
	if dismissal.CheckerName == "" {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "'checker_name' is required",
		}, w, nil)
		return
	}

	// confirm that the checker exists
	if _, ok := values.AllCheckerDefs[dismissal.CheckerName]; !ok {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    fmt.Sprintf("the checker '%s' does not exist", dismissal.CheckerName),
		}, w, nil)
		return
	}

	// validate level and identifiers for the level
	err := validateDismissLevel(dismissal)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
		}, w, nil)
		return
	}

	// validate that the cluster/node/bucket exist
	if dismissal.Level != values.AllDismissLevel {
		if err := m.validateContainerExists(dismissal.Dismissal); err != nil {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusBadRequest,
				Msg:    err.Error(),
			}, w, nil)
			return
		}
	}

	dismissal.ID = uuid.New().String()
	if err = m.store.AddDismissal(dismissal.Dismissal); err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not add dismissal",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}

func validateDismissLevel(dismissal dismissalRequest) error {
	// validate level and identifiers for the level
	switch dismissal.Level {
	case values.AllDismissLevel:
		// if dismissal level is all then no identifiers required
		if dismissal.ClusterUUID != "" || dismissal.BucketName != "" || dismissal.NodeUUID != "" ||
			dismissal.LogFile != "" {
			return fmt.Errorf("for level 0 no extra identifiers are requied")
		}
	case values.ClusterDismissLevel:
		// for cluster level only the cluster uuid is required
		if dismissal.ClusterUUID == "" {
			return fmt.Errorf("for level 1 'cluster_uuid' is required")
		}

		if dismissal.BucketName != "" || dismissal.NodeUUID != "" || dismissal.LogFile != "" {
			return fmt.Errorf("for level 1 only 'cluster_uuid' identifier is required")
		}
	case values.BucketDismissLevel:
		// for bucket level both cluster and bucket name required
		if dismissal.ClusterUUID == "" || dismissal.BucketName == "" {
			return fmt.Errorf("for level 2 'cluster_uuid' and 'bucket_name' identifiers are required")
		}

		if dismissal.NodeUUID != "" || dismissal.LogFile != "" {
			return fmt.Errorf("for level 2 only 'cluster_uuid' and 'bucket_name' identifiers are required")
		}
	case values.NodeDismissLevel:
		// for node level both cluster and node uuid required
		if dismissal.ClusterUUID == "" || dismissal.NodeUUID == "" {
			return fmt.Errorf("for level 3 'cluster_uuid' and 'node_uuid' identifiers are required")
		}

		if dismissal.BucketName != "" || dismissal.LogFile != "" {
			return fmt.Errorf("for level 3 only 'cluster_uuid' and 'node_uuid' identifiers are required")
		}
	case values.FileDismissLevel:
		// for node level cluster, node uuid  and file required
		if dismissal.ClusterUUID == "" || dismissal.NodeUUID == "" || dismissal.LogFile == "" {
			return fmt.Errorf("for level 4 'cluster_uuid', 'node_uuid' and 'log_file' identifiers are required")
		}

		if dismissal.BucketName != "" {
			return fmt.Errorf("for level 4 'bucket_name' cannot be provided")
		}
	default:
		return fmt.Errorf("invalid value for level, valid levels are 0 to 4 inclusive [0 - All, 1 - Cluster, " +
			"2 - Bucket, 3 - Node, 4 - File]")
	}

	return nil
}

func (m *Manager) validateContainerExists(dismissal values.Dismissal) error {
	cluster, err := m.store.GetCluster(dismissal.ClusterUUID, false)
	if err != nil {
		if errors.Is(err, values.ErrNotFound) {
			return fmt.Errorf("no cluster with UUID is '%s'", dismissal.ClusterUUID)
		}

		return fmt.Errorf("could not confirm that cluster exists. %w", err)
	}

	switch dismissal.Level {
	case values.BucketDismissLevel:
		for _, bucket := range cluster.BucketsSummary {
			if bucket.Name == dismissal.BucketName {
				return nil
			}
		}

		return fmt.Errorf("unknown bucket '%s'", dismissal.BucketName)
	case values.NodeDismissLevel:
		for _, node := range cluster.NodesSummary {
			if node.NodeUUID == dismissal.NodeUUID {
				return nil
			}
		}

		return fmt.Errorf("unknown node '%s'", dismissal.NodeUUID)
	}

	return nil
}

func (m *Manager) getDismissals(w http.ResponseWriter, r *http.Request) {
	dismissals, err := m.store.GetDismissals(getSearchSpaceFromQuery(r.URL.Query()))
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get dismissals from store",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	restutil.MarshalAndSend(http.StatusOK, dismissals, w, nil)
}

func (m *Manager) deleteDismissals(w http.ResponseWriter, r *http.Request) {
	err := m.store.DeleteDismissals(getSearchSpaceFromQuery(r.URL.Query()))
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not delete dismissals",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}

func (m *Manager) deleteDismissal(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	err := m.store.DeleteDismissals(values.DismissalSearchSpace{ID: &id})
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not delete dismissal",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}

func getSearchSpaceFromQuery(query url.Values) values.DismissalSearchSpace {
	var searchSpace values.DismissalSearchSpace
	if cluster := query.Get("cluster"); cluster != "" {
		searchSpace.ClusterUUID = &cluster
	}

	if bucket := query.Get("bucket"); bucket != "" {
		searchSpace.BucketName = &bucket
	}

	if node := query.Get("node"); node != "" {
		searchSpace.NodeUUID = &node
	}

	if file := query.Get("file"); file != "" {
		searchSpace.LogFile = &file
	}

	if checker := query.Get("checker"); checker != "" {
		searchSpace.CheckerName = &checker
	}

	return searchSpace
}
