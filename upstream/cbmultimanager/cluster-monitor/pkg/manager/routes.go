// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/manager/internal"
	embeddedui "github.com/couchbaselabs/cbmultimanager/ui"
)

func NewRouter(m *Manager) *mux.Router {
	r := mux.NewRouter()

	r.Use(m.initializedMiddleware)
	r.Use(m.authMiddleware)
	r.Use(loggingMiddleware)

	metricsAPI(r)

	if m.config.EnableAdminAPI {
		adminAPI(r, m)
	}

	if m.config.EnableClusterAPI {
		clusterAPI(r, m)
	}

	if m.config.EnableExtendedAPI {
		extendedAPI(r, m)
	}

	if m.config.UIRoot != "" {
		ui(r, m)
	}

	return r
}

func adminAPI(r *mux.Router, m *Manager) {
	v1 := r.PathPrefix("/api/v1").Subrouter()

	// Administration and auth endpoints.
	// Get initialization state.
	v1.HandleFunc("/self", m.getInitState).Methods("GET")
	// Initialize cbmultimanager.
	v1.HandleFunc("/self", m.initializeCluster).Methods("POST")
	// Create JWT token endpoint.
	v1.HandleFunc("/self/token", m.tokenLogin).Methods("POST")

	zap.S().Info("(Routes) Set up Admin API")
}

func metricsAPI(r *mux.Router) {
	v1 := r.PathPrefix("/api/v1").Subrouter()

	// Collects prometheus metrics.
	v1.Handle("/_prometheus", promhttp.Handler()).Methods("GET")
	// Provide standard endpoint to simplify configuration.
	v1.Handle("/metrics", promhttp.Handler()).Methods("GET")

	zap.S().Info("(Routes) Set up Metrics API")
}

func clusterAPI(r *mux.Router, m *Manager) {
	v1 := r.PathPrefix("/api/v1").Subrouter()

	// Cluster management related endpoints.
	// Gets all the clusters.
	v1.HandleFunc("/clusters", m.getClusters).Methods("GET")
	// Adds a new cluster.
	v1.HandleFunc("/clusters", m.addNewCluster).Methods("POST")

	// Get only one specific cluster.
	v1.HandleFunc("/clusters/{uuid}", m.getCluster).Methods("GET")
	// Used to update the user, password, certificate or give a new bootstrap host.
	v1.HandleFunc("/clusters/{uuid}", m.updateClusterInfo).Methods("PATCH")
	// Stops tracking the cluster.
	v1.HandleFunc("/clusters/{uuid}", m.deleteCluster).Methods("DELETE")

	zap.S().Info("(Routes) Set up Cluster Management API")
}

func extendedAPI(r *mux.Router, m *Manager) {
	v1 := r.PathPrefix("/api/v1").Subrouter()

	// Checker result endpoints.
	// Gets all checker results for the given cluster. Optionally filtered by node UUID or bucket name using query
	// parameters (bucket, node) respectively.
	v1.HandleFunc("/clusters/{uuid}/status", m.getClusterStatusReport).Methods("GET")
	// Triggers a heart beat and the running of all checkers for the specified cluster.
	v1.HandleFunc("/clusters/{uuid}/refresh", m.runChecksForCluster).Methods("POST")
	// Gets the result for a specific checker with the given name for the given cluster. Optionally filtered by node
	// UUID or bucket name using query parameters (bucket, node) respectively.
	v1.HandleFunc("/clusters/{uuid}/status/{name}", m.getClusterStatusCheckerResult).Methods("GET")
	// Get remote clusters of a cluster
	v1.HandleFunc("/clusters/{uuid}/remoteClusters", m.getClusterRemoteClusters).Methods("GET")
	// Get connections fo a cluster and bucket.
	v1.HandleFunc("/clusters/{uuid}/connections", m.getClusterConnections).Methods("GET")

	// Get a single node's details (unblocker for https://issues.couchbase.com/browse/CMOS-188)
	v1.HandleFunc("/clusters/{uuid}/node/{node_uuid}", m.getClusterNodeDetails).Methods("GET")

	// Endpoints to manage cluster aliases.
	// Add alias endpoint.
	v1.HandleFunc("/aliases/{alias}", m.AddAlias).Methods("POST")
	// Delete alias endpoint.
	v1.HandleFunc("/aliases/{alias}", m.DeleteAlias).Methods("DELETE")

	// Get status checker definition endpoints.
	// Get all status checkers definitions.
	v1.HandleFunc("/checkers", getStatusCheckerDefinitions).Methods("GET")
	// Get status checker definition for checker "name".
	v1.HandleFunc("/checkers/{name}", getStatusCheckerDefinition).Methods("GET")

	// Dismissal endpoints.
	// Add new dismissal.
	v1.HandleFunc("/dismissals", m.dismiss).Methods("POST")
	// Get all dismissals. Filterable by node, cluster, checker and/or file.
	v1.HandleFunc("/dismissals", m.getDismissals).Methods("GET")
	// Delete all dismissals that match query parameters. Filterable by node, cluster, checker and/or file.
	v1.HandleFunc("/dismissals", m.deleteDismissals).Methods("DELETE")
	// Get a dismissal by id.
	v1.HandleFunc("/dismissals/{id}", m.deleteDismissal).Methods("DELETE")

	// Manually trigger the status checks for all the clusters.
	v1.HandleFunc("/statusChecks/api", m.triggerAPIChecks).Methods("POST")

	// Force janitor to clean up stale data.
	v1.HandleFunc("/cleanup", m.cleanup).Methods("POST")

	// Endpoint to retrieve logs from the cluster.
	v1.HandleFunc("/clusters/{uuid}/nodes/{nodeUUID}/logs/{logName}", m.getLogs).Methods("GET")

	// Couchbase Cloud Endpoints
	v1.HandleFunc("/cloud/credentials", m.listCloudCreds).Methods("GET")
	v1.HandleFunc("/cloud/credentials", m.addCloudCreds).Methods("POST")

	v1.HandleFunc("/cloud/clusters", m.getCloudClusters).Methods("GET")
	v1.HandleFunc("/cloud/clusters/{id}", m.getCloudClusterStatus).Methods("GET")

	zap.S().Info("(Routes) Set up Extended API")
}

func determineUIHandler(uiRoot string) (http.Handler, error) {
	if uiRoot != "" {
		rootStat, err := os.Stat(uiRoot)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to stat ui-root %s: %w", uiRoot, err)
		}
		if err == nil {
			if !rootStat.IsDir() {
				return nil, fmt.Errorf("invalid ui-root %s: not a directory", uiRoot)
			}
			zap.S().Debugw("(Routes) Serving UI from file system", "dir", uiRoot)
			return internal.ServeStaticSite(http.Dir(uiRoot)), nil
		}
	}

	if embeddedui.EmbedPresent {
		zap.S().Debug("(Routes) Serving embedded UI")
		return internal.ServeStaticSite(http.FS(embeddedui.EmbeddedUI)), nil
	}

	return nil, nil
}

// ui serves the UI files, either from the --ui-root passed on the CLI, or the embedded files (if present).
// UI paths start with /ui. If the requested path exists, is a file, and is not a hidden file, ui will serve it,
// otherwise it will serve index.html (with the assumption that the UI will handle the sub-path)
func ui(r *mux.Router, m *Manager) {
	handler, err := determineUIHandler(m.config.UIRoot)
	if err != nil {
		zap.S().Errorw("(Routes) Failed to determine UI serving method, not serving UI", "error", err)
		return
	}
	if handler == nil {
		zap.S().Warn("(Routes) No --ui-root set and no embedded UI present, not serving UI.")
		return
	}

	r.PathPrefix(PathUIRoot).Methods("GET").Handler(http.StripPrefix(PathUIRoot, handler))
	r.Path("/").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, PathUIRoot, http.StatusSeeOther)
	})

	zap.S().Infow("(Routes) Set up UI")
}
