package manager

import (
	"fmt"

	"github.com/couchbaselabs/cbmultimanager/values"

	"github.com/couchbase/tools-common/connstr"
	"github.com/couchbase/tools-common/netutil"
	"go.uber.org/zap"
)

// getClusterStatusesFilterDismissed get the cluster status checkers but ignores any that is dismissed.
func (m *Manager) getClusterStatusesFilterDismissed(search values.CheckerSearch) ([]*values.WrappedCheckerResult,
	int, error) {
	// get dismissals for this cluster
	dismissals, err := m.store.GetDismissals(values.DismissalSearchSpace{ClusterUUID: search.Cluster})
	if err != nil {
		zap.S().Errorw("(Manager) Could not get dismissals", "cluster", *search.Cluster, "err", err)
	}

	// cannot take addresses of constants so have to do this hack
	allDismissLevel := values.AllDismissLevel
	// also get things dismissed at the all level
	allDismiss, err := m.store.GetDismissals(values.DismissalSearchSpace{Level: &allDismissLevel})
	if err != nil {
		zap.S().Errorw("(Manager) Could not get dismissals at all level", "err", err)
	}

	dismissals = append(dismissals, allDismiss...)

	results, err := m.store.GetCheckerResult(search)
	if err != nil {
		return nil, 0, fmt.Errorf("could not get checker results: %w", err)
	}

	filtered := make([]*values.WrappedCheckerResult, 0)
	for _, result := range results {
		var dismissed bool
		// go through the dismissals and if there then just increment the dismissal counter
		for _, dismissal := range dismissals {
			if dismissal.IsDismissed(result) {
				dismissed = true
				break
			}
		}

		if dismissed {
			continue
		}

		filtered = append(filtered, result)
	}

	return filtered, len(results) - len(filtered), nil
}

func resolveAddressesToSlice(resolvedHosts *connstr.ResolvedConnectionString) []string {
	scheme := "http"
	if resolvedHosts.UseSSL {
		scheme = "https"
	}

	addresses := make([]string, 0, len(resolvedHosts.Addresses))
	for _, address := range resolvedHosts.Addresses {
		addresses = append(addresses, fmt.Sprintf("%s://%s:%d", scheme, netutil.ReconstructIPV6(address.Host),
			address.Port))
	}

	return addresses
}
