// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package runner

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/bootstrap"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/store"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

type checkerFn func(self *bootstrap.Node) *values.WrappedCheckerResult

type Runner struct {
	checkers  map[string]checkerFn
	frequency time.Duration
	store     *store.InMemory
	self      *bootstrap.Node
}

func NewRunner(self *bootstrap.Node, frequency time.Duration, store *store.InMemory) *Runner {
	return &Runner{
		frequency: frequency,
		store:     store,
		checkers:  getSystemCheckers(),
		self:      self,
	}
}

func (r *Runner) Start(ctx context.Context, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		// Run on start up as well
		r.runSystemChecks()

		ticker := time.NewTicker(r.frequency)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.runSystemChecks()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (r *Runner) runSystemChecks() {
	for name, checkerFN := range r.checkers {
		zap.S().Infow("(Runner) Running system check", "name", name)
		res := checkerFN(r.self)
		if res.Error != nil {
			zap.S().Infow("(Runner) Checker failed", "name", name, "err", res.Error)
		}

		r.store.SetCheckerResult(name, res)
	}
}
