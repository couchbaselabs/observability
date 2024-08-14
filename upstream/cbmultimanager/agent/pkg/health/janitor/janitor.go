// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package janitor

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/store"
)

// Janitor cleans up old log checker results.
// It periodically removes all checker results with a log file that are older than a configurable cutoff (default 1h).
type Janitor struct {
	logger   *zap.SugaredLogger
	store    *store.InMemory
	interval time.Duration
	cutoff   time.Duration
}

func NewJanitor(store *store.InMemory, interval, cutoff time.Duration) *Janitor {
	return &Janitor{
		logger:   zap.S().Named("Janitor"),
		store:    store,
		interval: interval,
		cutoff:   cutoff,
	}
}

func (j *Janitor) Start(ctx context.Context, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		j.run()
		timer := time.NewTimer(j.interval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				j.run()
				timer.Reset(j.interval)
			}
		}
	}()
}

func (j *Janitor) run() {
	j.logger.Debug("Janitor running")
	defer j.logger.Debug("Janitor done.")
	for name, result := range j.store.GetCheckers() {
		if result.LogFile != "" && time.Since(result.Result.Time) > j.cutoff {
			j.store.RemoveCheckerResult(name)
		}
	}
}
