// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package runner

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/bootstrap"
	"github.com/couchbaselabs/cbmultimanager/agent/pkg/health/store"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

func TestRunner(t *testing.T) {
	s := store.NewInMemoryStore()

	frequency := 50 * time.Millisecond
	runner := NewRunner(&bootstrap.Node{}, frequency, s)

	var (
		c1Called int
		c2Called int
		c3Called int
	)

	start := time.Now().UTC()

	results := map[string]*values.WrappedCheckerResult{
		"c-1": {
			Result: &values.CheckerResult{
				Name:   "c-1",
				Status: values.GoodCheckerStatus,
				Time:   start,
			},
		},
		"c-2": {
			Result: &values.CheckerResult{
				Name:        "c-2",
				Status:      values.WarnCheckerStatus,
				Time:        start,
				Remediation: "Something is messed up",
			},
		},
		"c-3": {
			Result: &values.CheckerResult{
				Name:   "c-3",
				Status: values.MissingCheckerStatus,
				Time:   start,
			},
			Error: assert.AnError,
		},
	}

	runner.checkers = map[string]checkerFn{
		"c-1": func(self *bootstrap.Node) *values.WrappedCheckerResult {
			c1Called++
			return results["c-1"]
		},
		"c-2": func(self *bootstrap.Node) *values.WrappedCheckerResult {
			c2Called++
			return results["c-2"]
		},
		"c-3": func(self *bootstrap.Node) *values.WrappedCheckerResult {
			c3Called++
			return results["c-3"]
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	runner.Start(ctx, &wg)

	time.Sleep(frequency * 3)
	cancel()

	require.GreaterOrEqual(t, c1Called, 3)
	require.GreaterOrEqual(t, c2Called, 3)
	require.GreaterOrEqual(t, c3Called, 3)

	checkers := s.GetCheckers()
	require.Equal(t, results, checkers)
}
