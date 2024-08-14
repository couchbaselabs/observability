// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package status

import (
	"context"
	"time"

	"github.com/couchbase/tools-common/hofp"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/status/progress"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

type CheckExecutor struct {
	pool     *hofp.Pool
	checkers map[string]values.CheckerFn
	logger   *zap.SugaredLogger
	progress *progress.Monitor
}

func NewCheckExecutor(maxWorkers int) *CheckExecutor {
	return &CheckExecutor{
		pool: hofp.NewPool(hofp.Options{
			LogPrefix: "(Check Runner)",
			Size:      maxWorkers,
		}),
		checkers: allCheckerFns,
		logger:   zap.S().Named("Check Runner"),
		progress: progress.NewMonitor(),
	}
}

func (c *CheckExecutor) GetProgressFor(uuid string) (*values.ClusterProgress, error) {
	return c.progress.GetProgressFor(uuid)
}

func (c *CheckExecutor) CheckCluster(cluster values.CouchbaseCluster) (<-chan []*values.WrappedCheckerResult, error) {
	enqueued := time.Now()
	c.progress.ClusterRunStart(cluster.UUID, len(c.checkers))
	//nolint:errcheck
	defer c.progress.ClusterRunEnd(cluster.UUID)

	result := make(chan []*values.WrappedCheckerResult, 1)
	err := c.pool.Queue(func(ctx context.Context) error {
		start := time.Now()
		defer func() {
			c.logger.Debugw("Checker run complete.", "elapsed", time.Since(start))
		}()
		c.logger.Debugw("Starting checker run", "cluster", cluster.UUID, "spentInQueue", start.Sub(enqueued))
		retVal := make([]*values.WrappedCheckerResult, 0)
		for name, checker := range c.checkers {
			res := versionCheck(name, &cluster, values.AllCheckerDefs)
			if res != nil {
				retVal = append(retVal, res)
				continue
			}
			c.logger.Debugw("Running checker", "checker", name, "cluster", cluster.UUID)
			result, err := checker(cluster)
			if err != nil {
				c.logger.Warnw("Error running checker", "checker", name, "cluster", cluster.UUID, "error", err)
			}
			retVal = append(retVal, result...)
			c.progress.CheckerDone(cluster.UUID, err != nil) //nolint:errcheck
		}
		result <- retVal
		close(result)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *CheckExecutor) Stop() error {
	return c.pool.Stop()
}
