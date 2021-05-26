package manager

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/couchbaselabs/cbmultimanager/status"
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/restutil"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type resultCluster struct {
	UUID           string                `json:"uuid"`
	Name           string                `json:"name"`
	NodesSummary   values.NodesSummary   `json:"nodes_summary"`
	BucketsSummary values.BucketsSummary `json:"buckets_summary"`
	HeartBeatIssue values.HeartIssue     `json:"heart_beat_issue,omitempty"`
	LastUpdate     time.Time             `json:"last_update"`

	StatusResults  []*values.WrappedCheckerResult `json:"status_results"`
	StatusProgress *values.ClusterProgress        `json:"status_progress"`
	Dismissed      int                            `json:"dismissed"`
}

func (m *Manager) getClusterStatusReport(w http.ResponseWriter, r *http.Request) {
	m.getCheckerResultCommon(w, r, true)
}

func (m *Manager) getClusterStatusCheckerResult(w http.ResponseWriter, r *http.Request) {
	m.getCheckerResultCommon(w, r, false)
}

func (m *Manager) getCheckerResultCommon(w http.ResponseWriter, r *http.Request, filterDismissed bool) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	search := values.CheckerSearch{Cluster: &uuid}
	if name, ok := vars["name"]; ok {
		search.Name = &name
	}

	// get optional filters
	if node := r.URL.Query().Get("node"); node != "" {
		search.Node = &node
	}

	if bucket := r.URL.Query().Get("bucket"); bucket != "" {
		search.Bucket = &bucket
	}

	cluster, err := m.store.GetCluster(uuid, false)
	if err != nil {
		if errors.Is(err, values.ErrNotFound) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusNotFound,
				Msg:    fmt.Sprintf("cluster with UUID '%s' not found", uuid),
			}, w, nil)
			return
		}

		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get cluster details",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	clusterOut := &resultCluster{
		UUID:           cluster.UUID,
		Name:           cluster.Name,
		BucketsSummary: cluster.BucketsSummary,
		NodesSummary:   cluster.NodesSummary,
		HeartBeatIssue: cluster.HeartBeatIssue,
		LastUpdate:     cluster.LastUpdate,
	}

	if filterDismissed {
		clusterOut.StatusResults, clusterOut.Dismissed, err = m.getClusterStatusesFilterDismissed(search)
	} else {
		clusterOut.StatusResults, err = m.store.GetCheckerResult(search)
	}

	if err != nil {
		return
	}

	// we do not care about this error
	clusterOut.StatusProgress, _ = m.statusMonitor.GetProgressFor(cluster.UUID)

	restutil.MarshalAndSend(http.StatusOK, clusterOut, w, nil)
}

func getStatusCheckerDefinitions(w http.ResponseWriter, _ *http.Request) {
	restutil.MarshalAndSend(http.StatusOK, status.AllCheckerDefs, w, nil)
}

func getStatusCheckerDefinition(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	val, ok := status.AllCheckerDefs[name]
	if !ok {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusNotFound,
			Msg:    "no checker with name: " + name,
		}, w, nil)
		return
	}

	restutil.MarshalAndSend(http.StatusOK, &val, w, nil)
}

func (m *Manager) triggerAPIChecks(w http.ResponseWriter, _ *http.Request) {
	if err := m.statusMonitor.TriggerAPICheck(nil); err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not trigger check",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}

func (m *Manager) runChecksForCluster(w http.ResponseWriter, r *http.Request) {
	uuid := mux.Vars(r)["uuid"]

	cluster, err := m.store.GetCluster(uuid, true)
	if err != nil {
		if errors.Is(err, values.ErrNotFound) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusNotFound,
				Msg:    "cluster not found",
			}, w, nil)
			return
		}

		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not retrieve cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	go func() {
		zap.S().Infow("(Manager) Starting force refresh of cluster", "cluster", uuid)

		if err = m.heartMonitor.HeartBeatCluster(cluster); err != nil {
			zap.S().Errorw("(Manager) Error with forced heart beat", "cluster", cluster.UUID, "err", err)
			return
		}

		// we do a heartbeat so we have to refresh the cluster
		cluster, err = m.store.GetCluster(uuid, true)
		if err != nil {
			zap.S().Errorw("(Manager) Could not get refreshed cluster", "cluster", uuid, "err", err)
			return
		}

		if err = m.statusMonitor.TriggerAPICheck(cluster); err != nil {
			zap.S().Errorw("(Manager) Could not trigger API checks for cluster", "cluster", uuid, "err", err)
		}
	}()

	// we are not actually going to wait until the checks are run to send a response
	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}
