// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/connstr"
	"github.com/couchbase/tools-common/netutil"
	"github.com/couchbase/tools-common/restutil"
	"go.uber.org/zap"
)

// getClusterStatusesFilterDismissed get the cluster status checkers but ignores any that is dismissed.
func (m *Manager) getClusterStatusesFilterDismissed(search values.CheckerSearch) ([]*values.WrappedCheckerResult,
	int, error,
) {
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

// convertAliasToUUID is a utility to decode aliases, if it fails to decode it it will send an error status code back.
func (m *Manager) convertAliasToUUID(aliasName string, w http.ResponseWriter) (string, bool) {
	// It is not an alias so just return
	if !strings.HasPrefix(aliasName, aliasPrefix) {
		return aliasName, true
	}

	alias, err := m.store.GetAlias(aliasName)
	if err == nil {
		return alias.ClusterUUID, true
	}

	if errors.Is(err, values.ErrNotFound) {
		restutil.HandleErrorWithExtras(restutil.ErrorResponse{
			Status: http.StatusNotFound,
			Msg:    "could not find cluster with alias: " + aliasName,
		}, w, nil)
		return "", false
	}

	zap.S().Warnw("(Manager) Error converting alias to UUID", "err", err)
	restutil.HandleErrorWithExtras(restutil.ErrorResponse{
		Status: http.StatusInternalServerError,
		Msg:    fmt.Sprintf("could not convert alias '%s' to uuid", aliasName),
		Extras: err.Error(),
	}, w, nil)
	return "", false
}
