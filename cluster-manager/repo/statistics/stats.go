package statistics

import (
	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	statusVar = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "checker_status",
		Help: "Checker results",
	}, []string{"status", "cluster", "node", "bucket", "name"},
	)
	statusErr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "checker_errored",
		Help: "Checkers which have failed to run",
	}, []string{"name"})
)

// CheckStatus increments prometheus metrics based on checker result.
func CheckStatus(results []*values.WrappedCheckerResult) {
	for _, item := range results {
		if item.Result == nil {
			continue
		}

		if item.Error != nil {
			statusErr.WithLabelValues(item.Result.Name).Inc()
			continue
		}

		statusVar.WithLabelValues(string(item.Result.Status), item.Cluster, item.Node, item.Bucket,
			item.Result.Name).Inc()
	}
}

// CollectStats registers prometheus stats.
func CollectStats() {
	prometheus.MustRegister(statusVar, statusErr)
}
