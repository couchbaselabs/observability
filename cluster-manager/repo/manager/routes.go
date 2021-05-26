package manager

import (
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(m *Manager) *mux.Router {
	r := mux.NewRouter()

	r.Use(m.initializedMiddleware)
	r.Use(m.authMiddleware)
	r.Use(loggingMiddleware)

	v1API(r, m)
	ui(r, m)

	return r
}

func v1API(r *mux.Router, m *Manager) {
	v1 := r.PathPrefix("/api/v1").Subrouter()

	// Administration and auth endpoints.
	// Get initialization state.
	v1.HandleFunc("/self", m.getInitState).Methods("GET")
	// Initialize cbmulitmanager.
	v1.HandleFunc("/self", m.initializeCluster).Methods("POST")
	// Create JWT token endpoint.
	v1.HandleFunc("/self/token", m.tokenLogin).Methods("POST")

	// Collects prometheus metrics.
	v1.Handle("/_prometheus", promhttp.Handler()).Methods("GET")

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

	// Checker result endpoints.
	// Gets all checker results for the given cluster. Optionally filtered by node UUID or bucket name using query
	// parameters (bucket, node) respectively.
	v1.HandleFunc("/clusters/{uuid}/status", m.getClusterStatusReport).Methods("GET")
	// Triggers a heart beat and the running of all checkers for the specified cluster.
	v1.HandleFunc("/clusters/{uuid}/refresh", m.runChecksForCluster).Methods("POST")
	// Gets the result for a specific checker with the given name for the given cluster. Optionally filtered by node
	// UUID or bucket name using query parameters (bucket, node) respectively.
	v1.HandleFunc("/clusters/{uuid}/status/{name}", m.getClusterStatusCheckerResult).Methods("GET")
	// Get connections fo a cluster and bucket.
	v1.HandleFunc("/clusters/{uuid}/connections", m.getClusterConnections).Methods("GET")

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
}

// ui serves the UI files under the UIRoot passed in the CLI. UI paths start with /ui or /static. Worth noting that
// requests to / or /ui/... will be re-directed to index.html as the front end code handles transitions in the frontend.
func ui(r *mux.Router, m *Manager) {
	r.PathPrefix("/static").Handler(http.StripPrefix("/static", http.FileServer(http.Dir(m.config.UIRoot))))
	r.PathPrefix("/ui").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(m.config.UIRoot, "index.html"))
	})
	r.Path("/").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(m.config.UIRoot, "index.html"))
	})
}
