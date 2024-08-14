// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/memcached"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/connstr"
	"github.com/couchbase/tools-common/restutil"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func (m *Manager) getClusters(w http.ResponseWriter, _ *http.Request) {
	clusters, err := m.store.GetClusters(false, false)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get clusters",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	// get cluster summary
	for _, cluster := range clusters {
		// CE clusters dont run checkers so no need to try and get a summary.
		if !cluster.Enterprise {
			continue
		}

		cluster.StatusSummary, err = m.getStatusSummaryForCluster(cluster.UUID, true)
		if err != nil {
			zap.S().Warnw("(Manager) Failed to get cluster summary for cluster", "cluster", cluster.UUID, "err", err)
		}
	}

	restutil.MarshalAndSend(http.StatusOK, clusters, w, nil)
}

// getStatusSummaryForCluster builds the status summary taking into account dismissals
func (m *Manager) getStatusSummaryForCluster(clusterUUID string, filterDismissed bool) (*values.ClusterStatusSummary,
	error,
) {
	var (
		results   []*values.WrappedCheckerResult
		dismissed int
		err       error
	)

	if filterDismissed {
		results, dismissed, err = m.getClusterStatusesFilterDismissed(values.CheckerSearch{Cluster: &clusterUUID})
	} else {
		results, err = m.store.GetCheckerResult(values.CheckerSearch{Cluster: &clusterUUID})
	}

	if err != nil {
		return nil, err
	}

	summary := values.ClusterStatusSummary{Dismissed: uint64(dismissed)}
	for _, result := range results {
		switch result.Result.Status {
		case values.GoodCheckerStatus:
			summary.Good++
		case values.WarnCheckerStatus:
			summary.Warnings++
		case values.AlertCheckerStatus:
			summary.Alerts++
		case values.InfoCheckerStatus:
			summary.Info++
		case values.MissingCheckerStatus:
			summary.Missing++
		}
	}

	return &summary, nil
}

func (m *Manager) getCluster(w http.ResponseWriter, r *http.Request) {
	uuid, ok := m.convertAliasToUUID(mux.Vars(r)["uuid"], w)
	if !ok {
		return
	}

	cluster, err := m.store.GetCluster(uuid, false)
	if err != nil {
		if errors.Is(err, values.ErrNotFound) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusNotFound,
				Msg:    "could not find cluster with uuid: " + uuid,
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

	// CE clusters don't run checkers so can ignore everything after this point.
	if !cluster.Enterprise {
		restutil.MarshalAndSend(http.StatusOK, cluster, w, nil)
		return
	}

	// get cluster summary
	cluster.StatusSummary, err = m.getStatusSummaryForCluster(cluster.UUID, true)
	if err != nil {
		zap.S().Warnw("(Manager) Failed to get cluster summary for cluster", "cluster", cluster.UUID, "err", err)
	}

	// we do not care about this error. It will only be present before the first run so that is fine.
	if cm, ok := m.clusterManagers.Load(cluster.UUID); ok {
		cluster.StatusProgress, _ = cm.GetProgress()
	}

	restutil.MarshalAndSend(http.StatusOK, cluster, w, nil)
}

func (m *Manager) deleteCluster(w http.ResponseWriter, r *http.Request) {
	uuid, ok := m.convertAliasToUUID(mux.Vars(r)["uuid"], w)
	if !ok {
		return
	}

	if cm, ok := m.clusterManagers.Load(uuid); ok {
		cm.Stop()
		m.clusterManagers.Remove(uuid)
	}

	err := m.store.DeleteDismissals(values.DismissalSearchSpace{ClusterUUID: &uuid})
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not delete dismissals for cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	if err = m.store.DeleteCheckerResults(values.CheckerSearch{Cluster: &uuid}); err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not delete cluster check results",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	if err = m.store.DeleteCluster(uuid); err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not delete cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	zap.S().Infow("(Manager) Cluster deleted", "cluster", uuid)
	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}

type addClusterReq struct {
	Host     string `json:"host"`
	User     string `json:"user"`
	Password string `json:"password"`
	Alias    string `json:"alias"`

	CaCert []byte `json:"ca_cert"`
}

func (m *Manager) addNewCluster(w http.ResponseWriter, r *http.Request) {
	var req addClusterReq
	if !restutil.DecodeJSONRequestBody(r.Body, &req, w) {
		return
	}

	// validate all mandatory fields are provided
	if len(req.Host) == 0 {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "host is required",
		}, w, nil)
		return
	}

	if len(req.User) == 0 {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "user is required",
		}, w, nil)
		return
	}

	if len(req.Password) == 0 {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "password is required",
		}, w, nil)
		return
	}

	if len(req.Alias) > 100 {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "Maximum alias length is 100 characters",
		}, w, nil)
		return
	}

	if len(req.Alias) > 0 && !strings.HasPrefix(req.Alias, aliasPrefix) {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "Aliases must start with " + aliasPrefix,
		}, w, nil)
		return
	}

	// Get the SystemCertPool, continue with an empty pool on error
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// the AppendCertsFromPEM function checks that the bytes are a valid cert
	if req.CaCert != nil {
		if ok := rootCAs.AppendCertsFromPEM(req.CaCert); !ok {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusBadRequest,
				Msg:    "invalid certificate",
			}, w, nil)
			return
		}
	}

	// parse host
	hosts, err := connstr.Parse(req.Host)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "invalid host",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	resolvedHosts, err := hosts.Resolve()
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not resolve hosts",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	// create client to communicate with cluster
	// skip cacert verify if none given
	client, err := couchbase.NewClient(resolveAddressesToSlice(resolvedHosts), req.User, req.Password,
		&tls.Config{InsecureSkipVerify: req.CaCert == nil, RootCAs: rootCAs}, false)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not establish connection with remote cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	// if the client was created then we could communicate with the cluster and got the UUID as well as the nodes so we
	// also want to get the buckets summary at the start
	buckets, err := client.GetBucketsSummary()
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get bucket summary from cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	remoteClusters, err := client.GetRemoteClusters(client.ClusterInfo.ClusterName, client.ClusterInfo.ClusterUUID)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get remote clusters from cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	cluster := &values.CouchbaseCluster{
		UUID:             client.ClusterInfo.ClusterUUID,
		Enterprise:       client.GetClusterInfo().Enterprise,
		Name:             client.ClusterInfo.ClusterName,
		NodesSummary:     client.ClusterInfo.NodesSummary,
		ClusterInfo:      client.ClusterInfo.ClusterInfo,
		PoolsRaw:         client.ClusterInfo.PoolsRaw,
		DeveloperPreview: client.ClusterInfo.DeveloperPreview,
		User:             req.User,
		Password:         req.Password,
		HeartBeatIssue:   values.NoHeartIssue,
		CaCert:           req.CaCert,
		BucketsSummary:   buckets,
		RemoteClusters:   remoteClusters,
		Alias:            req.Alias,
	}

	if err = m.store.AddCluster(cluster); err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not save cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	zap.S().Infow("(Manager) Cluster added", "cluster", client.ClusterInfo.ClusterUUID)
	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)

	// CE clusters don't run checkers so skip triggering API check
	if !cluster.Enterprise {
		return
	}

	// Create a SingleClusterManager for it and start it going
	m.clusterManagers.Store(cluster.UUID, NewSingleClusterManager(
		cluster,
		client,
		m.store,
		m.alertmanager,
		m.checkExecutor,
		DefaultFrequencyConfiguration,
	))

	if cm, ok := m.clusterManagers.Load(cluster.UUID); ok {
		err := cm.Start()
		if err != nil {
			zap.S().Errorw("(Manager) Failed to start single cluster manager: %w", err)
		}
	}
}

func (m *Manager) updateClusterInfo(w http.ResponseWriter, r *http.Request) {
	uuid, ok := m.convertAliasToUUID(mux.Vars(r)["uuid"], w)
	if !ok {
		return
	}

	// before we do any fetches to the store we will verify the request body
	var req addClusterReq
	if !restutil.DecodeJSONRequestBody(r.Body, &req, w) {
		return
	}

	// TODO: add max length constraints to the user and password
	// the request must have at least one of host, user, password or cacert
	if req.CaCert == nil && req.User == "" && req.Password == "" && req.Host == "" {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "at least one of [host, user, password, cacert] is required",
		}, w, nil)
		return
	}

	// Get the SystemCertPool, continue with an empty pool on error
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// if a cacert is given then validate it
	if len(req.CaCert) > 0 {
		// the AppendCertsFromPEM function checks that the bytes are a valid cert, if ok == false then it wasn't valid
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "invalid certificate",
		}, w, nil)
		return
	}

	var hosts []string
	// if host given parse and validate
	if req.Host != "" {
		parsed, err := connstr.Parse(req.Host)
		if err != nil {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusBadRequest,
				Msg:    "invalid host",
				Extras: err.Error(),
			}, w, nil)
			return
		}

		resolved, err := parsed.Resolve()
		if err != nil {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusInternalServerError,
				Msg:    "could not resolve new hosts",
				Extras: err.Error(),
			}, w, nil)
			return
		}

		hosts = resolveAddressesToSlice(resolved)
	}

	// try and get cluster from store
	cluster, err := m.store.GetCluster(uuid, true)
	if err != nil {
		if errors.Is(err, values.ErrNotFound) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusNotFound,
				Msg:    "could not find cluster with uuid: " + uuid,
			}, w, nil)
			return
		}

		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "failed to retrieve cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	if hosts == nil {
		hosts = cluster.NodesSummary.GetHosts()
	}

	user := cluster.User
	if req.User != "" {
		user = req.User
	}

	password := cluster.Password
	if req.Password != "" {
		password = req.Password
	}

	useCert := len(req.CaCert) != 0 || len(cluster.CaCert) != 0
	if len(req.CaCert) == 0 && len(cluster.CaCert) != 0 {
		rootCAs.AppendCertsFromPEM(cluster.CaCert)
	}

	// confirm we can communicate with the cluster with the new information
	client, err := couchbase.NewClient(hosts, user, password, &tls.Config{
		InsecureSkipVerify: !useCert,
		RootCAs:            rootCAs,
	}, false)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not communicate with cluster using new information",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	// check that the cluster is still the same cluster, we can do this by checking the cluster UUID
	if client.ClusterInfo.ClusterUUID != cluster.UUID {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "new cluster information does not point to the same cluster",
			Extras: fmt.Sprintf("old uuid != new uuid. '%s' != '%s'", cluster.UUID, client.ClusterInfo.ClusterUUID),
		}, w, nil)
		return
	}

	// once all check pass do the update
	newCluster := &values.CouchbaseCluster{
		UUID:         cluster.UUID,
		Name:         client.ClusterInfo.ClusterName,
		NodesSummary: client.ClusterInfo.NodesSummary,
		Enterprise:   client.GetClusterInfo().Enterprise,
		User:         req.User,
		Password:     req.Password,
		CaCert:       req.CaCert,
	}
	err = m.store.UpdateCluster(newCluster)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "cluster update failed",
			Extras: err.Error(),
		}, w, nil)
		return
	}
	if cm, ok := m.clusterManagers.Load(cluster.UUID); ok {
		cm.UpdateClusterInfo(newCluster)
	}

	zap.S().Infow("(Manager) Cluster updated", "cluster", client.ClusterInfo.ClusterUUID)
	restutil.SendJSONResponse(http.StatusOK, []byte{}, w, nil)
}

func (m *Manager) getClusterRemoteClusters(w http.ResponseWriter, r *http.Request) {
	uuid, ok := m.convertAliasToUUID(mux.Vars(r)["uuid"], w)
	if !ok {
		return
	}

	cluster, err := m.store.GetCluster(uuid, true)
	if err != nil {
		if errors.Is(err, values.ErrNotFound) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusBadRequest,
				Msg:    "cluster does not exist",
			}, w, nil)
			return
		}

		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	restutil.MarshalAndSend(http.StatusOK, cluster.RemoteClusters, w, nil)
}

func (m *Manager) getClusterConnections(w http.ResponseWriter, r *http.Request) {
	uuid, ok := m.convertAliasToUUID(mux.Vars(r)["uuid"], w)
	if !ok {
		return
	}

	cluster, err := m.store.GetCluster(uuid, true)
	if err != nil {
		if errors.Is(err, values.ErrNotFound) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusBadRequest,
				Msg:    "cluster does not exist",
			}, w, nil)
			return
		}

		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	if !cluster.Enterprise {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusBadRequest,
			Msg:    "Can only get connections for Enterprise Edition clusters",
		}, w, nil)
		return
	}

	memD, err := memcached.NewMemcachedClient(cluster)
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not create memcached client",
			Extras: err.Error(),
		}, w, nil)
		return
	}
	defer memD.Close()

	connStats, err := memD.GetConnectionsFor()
	if err != nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get connections",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	restutil.MarshalAndSend(http.StatusOK, connStats, w, nil)
}

// getClusterNodeDetails is an unblocker for https://issues.couchbase.com/browse/CMOS-188
func (m *Manager) getClusterNodeDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterUUID, ok := m.convertAliasToUUID(vars["uuid"], w)
	if !ok {
		return
	}
	nodeUUID, ok := vars["node_uuid"]
	if !ok {
		return
	}

	cluster, err := m.store.GetCluster(clusterUUID, true)
	if err != nil {
		if errors.Is(err, values.ErrNotFound) {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusNotFound,
				Msg:    "cluster does not exist",
			}, w, nil)
			return
		}

		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusInternalServerError,
			Msg:    "could not get cluster",
			Extras: err.Error(),
		}, w, nil)
		return
	}

	var node *values.NodeSummary
	for i, test := range cluster.NodesSummary {
		if test.NodeUUID == nodeUUID {
			node = &cluster.NodesSummary[i]
			break
		}
	}
	if node == nil {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusNotFound,
			Msg:    "node not found",
		}, w, nil)
		return
	}

	restutil.MarshalAndSend(http.StatusOK, node, w, nil)
}
