// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/restutil"
	"github.com/couchbaselabs/couchbase-cloud-go-client/couchbasecloud"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func (m *Manager) addCloudCreds(w http.ResponseWriter, r *http.Request) {
	var creds *values.Credential
	if !restutil.DecodeJSONRequestBody(r.Body, &creds, w) {
		return
	}

	zap.S().Infow("(Manager) Adding cloud credentials", "name", creds.Name)
	if err := m.store.AddCloudCredentials(creds); err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not add credentials",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}

func (m *Manager) listCloudCreds(w http.ResponseWriter, _ *http.Request) {
	creds, err := m.store.GetCloudCredentials(false)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get credentials",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	restutil.MarshalAndSend(http.StatusOK, &creds, w, nil)
}

func (m *Manager) getCloudClusters(w http.ResponseWriter, r *http.Request) {
	cred, ok := m.getCloudCred(w)
	if !ok {
		return
	}

	pagination, err := getCloudPaginationParameters(r.URL.Query())
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    err.Error(),
		}, w, nil)
		return
	}

	cloudClient := couchbasecloud.NewClient(cred.AccessKey, cred.SecretKey)
	clusters, err := cloudClient.ListClusters(&couchbasecloud.ListClustersOptions{
		Page:      pagination.page,
		PerPage:   pagination.size,
		SortBy:    pagination.sortBy,
		CloudId:   pagination.cloudID,
		ProjectId: pagination.projectID,
	})
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get cloud clusters",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	restutil.MarshalAndSend(http.StatusOK, &clusters, w, nil)
}

// getCloudCred returns the first cloud cred if it exists if not it will return an error via HTTP and return nil, false.
func (m *Manager) getCloudCred(w http.ResponseWriter) (*values.Credential, bool) {
	creds, err := m.store.GetCloudCredentials(true)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get cloud credentials",
			Extras: err.Error(),
		}, w, nil)
		return nil, false
	}

	if len(creds) == 0 {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "no cloud credentials registered",
		}, w, nil)
		return nil, false
	}

	return creds[0], true
}

type cloudPagination struct {
	page      int
	size      int
	sortBy    *string
	cloudID   *string
	projectID *string
}

func getCloudPaginationParameters(query url.Values) (*cloudPagination, error) {
	var (
		pagination cloudPagination
		err        error
	)

	if pageStr := query.Get("page"); pageStr != "" {
		pagination.page, err = strconv.Atoi(pageStr)
		if err != nil {
			return nil, fmt.Errorf("invalid value '%s' for query parameter 'page'", pageStr)
		}
	}

	if sizeStr := query.Get("size"); sizeStr != "" {
		var err error
		pagination.size, err = strconv.Atoi(sizeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid value '%s' for query parameter 'size'", sizeStr)
		}
	}

	if sortBy := query.Get("sortBy"); sortBy != "" {
		pagination.sortBy = &sortBy
	}

	if cloudID := query.Get("cloudID"); cloudID != "" {
		pagination.cloudID = &cloudID
	}

	if projectID := query.Get("projectID"); projectID != "" {
		pagination.projectID = &projectID
	}
	return &pagination, nil
}

func (m *Manager) getCloudClusterStatus(w http.ResponseWriter, r *http.Request) {
	clusterID := mux.Vars(r)["id"]

	cred, ok := m.getCloudCred(w)
	if !ok {
		return
	}

	cloudClient := couchbasecloud.NewClient(cred.AccessKey, cred.SecretKey)
	status, err := cloudClient.GetClusterStatus(clusterID)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get cluster status",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	type cloudCluster struct {
		Status      string                      `json:"status"`
		Health      string                      `json:"health,omitempty"`
		BucketStats *couchbasecloud.BucketStats `json:"bucket_stats,omitempty"`
		NodeStats   *couchbasecloud.NodeStats   `json:"node_stats,omitempty"`
	}

	// Can only get health for ready clusters.
	if status.Status != "ready" {
		restutil.MarshalAndSend(http.StatusOK, &cloudCluster{Status: status.Status}, w, nil)
		return
	}

	health, err := cloudClient.GetClusterHealth(clusterID)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get cluster health",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	restutil.MarshalAndSend(http.StatusOK, &cloudCluster{
		Status:      health.Status,
		Health:      health.Health,
		BucketStats: health.BucketStats,
		NodeStats:   health.NodeStats,
	}, w, nil)
}
