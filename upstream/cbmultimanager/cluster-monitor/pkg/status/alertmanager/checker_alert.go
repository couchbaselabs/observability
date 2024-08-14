// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package alertmanager

import (
	"fmt"
	"time"

	"github.com/couchbase/tools-common/netutil"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/status/alertmanager/types"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

// checkerAlert wraps a WrappedCheckerResult with metadata needed for Alertmanager
type checkerAlert struct {
	result     *values.WrappedCheckerResult
	activeAt   time.Time
	resolvedAt *time.Time
	// NOTE: a change in checker status is really the same as firing a new alert for the same checker,
	// as alert labels are immutable - hence hanging on to it, rather than using result.Result.Status directly
	status values.CheckerStatus
	// we cache these to avoid needing to go to the store on every call to labels
	cachedClusterName  string
	cachedNodeHostname string
	baseLabels         map[string]string
}

func newCheckerAlert(result *values.WrappedCheckerResult, store storage.Store) (*checkerAlert, error) {
	if result.Result.Status == values.GoodCheckerStatus {
		return nil, fmt.Errorf("cannot create alert for good status")
	}
	cluster, err := store.GetCluster(result.Cluster, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster %s from store: %w", result.Cluster, err)
	}
	var nodeName string
	if result.Node != "" {
		for _, node := range cluster.NodesSummary {
			if node.NodeUUID == result.Node {
				nodeName = node.Host
				break
			}
		}
	}
	return &checkerAlert{
		result:             result,
		activeAt:           result.Result.Time,
		status:             result.Result.Status,
		cachedClusterName:  cluster.Name,
		cachedNodeHostname: nodeName,
	}, nil
}

// additional labels to be sent to Alertmanager.
func (c *checkerAlert) withBaseLabels(labels map[string]string) *checkerAlert {
	if c.baseLabels == nil {
		c.baseLabels = make(map[string]string)
	}

	for k, v := range labels {
		c.baseLabels[k] = v
	}

	return c
}

// labels determines the Alertmanager labels to send for this alert.
func (c *checkerAlert) labels() map[string]string {
	kind := "cluster"
	if c.result.Node != "" {
		kind = "node"
	}
	if c.result.Bucket != "" {
		kind = "bucket"
	}
	defn := values.AllCheckerDefs[c.result.Result.Name]
	// https://github.com/couchbaselabs/observability/blob/main/CONTRIBUTING.md#alerting-rules-prometheus-and-loki
	result := map[string]string{
		"job":               "couchbase_cluster_monitor",
		"kind":              kind,
		"severity":          c.status.Severity(),
		"health_check_id":   defn.ID,
		"health_check_name": c.result.Result.Name,
		"cluster_name":      c.cachedClusterName,
	}
	if c.cachedNodeHostname != "" {
		result["node"] = netutil.TrimSchema(c.cachedNodeHostname)
	}
	if c.result.Bucket != "" {
		result["bucket"] = c.result.Bucket
	}

	for k, v := range c.baseLabels {
		if result[k] == "" {
			result[k] = v
		}
	}

	return result
}

func (c *checkerAlert) annotations() map[string]string {
	defn := values.AllCheckerDefs[c.result.Result.Name]
	result := map[string]string{
		"summary":     defn.Title,
		"description": defn.Description,
		"remediation": c.result.Result.Remediation,
	}
	if len(c.result.Result.Value) > 0 {
		// TODO: ideally all the information would be in the above fields, rather than needing to look at the raw JSON
		result["raw_value"] = string(c.result.Result.Value)
	}
	return result
}

// alertCacheKey is the [name, cluster, node, bucket, severity] of the alert.
// This is an array, not a slice, because arrays are comparable and hashable.
// This should never be constructed manually, instead use checkerAlert.cacheKey()
type alertCacheKey [5]string

// cacheKey returns an array of the elements that make an alert unique (i.e. the ones that determine
// its labels' values).
func (c *checkerAlert) cacheKey() alertCacheKey {
	return [5]string{
		c.result.Result.Name,
		c.cachedClusterName,
		c.cachedNodeHostname,
		c.result.Bucket,
		c.status.Severity(),
	}
}

func (c *checkerAlert) asPostableAlert() types.PostableAlert {
	return types.PostableAlert{
		Labels:      c.labels(),
		Annotations: c.annotations(),
		StartsAt:    &c.activeAt,
		EndsAt:      c.resolvedAt,
	}
}
