// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package internal

import (
	"testing"

	"go.uber.org/atomic"
)

// concurrencyGuard is a testing helper that ensures it is not called by more than a given number of callers
// concurrently. To use it:
//
// 1. Create a concurrencyGuard, specifying the maximum number of concurrent callers permitted
// 2. In each concurrent caller, add the following code:
//
//	guard.Start()
//	defer guard.Stop()
//
// The guard will keep track of the number of calls to Start and Stop, and ensure that no more than the given maximum
// can call Start before Stop (i.e. are concurrently executing).
type concurrencyGuard struct {
	t     *testing.T
	max   uint32
	value *atomic.Int32
}

func newConcurrencyGuard(t *testing.T, max uint32) *concurrencyGuard {
	return &concurrencyGuard{
		t:     t,
		max:   max,
		value: atomic.NewInt32(0),
	}
}

func (g *concurrencyGuard) Start() {
	newVal := g.value.Inc()
	if uint32(newVal) > g.max {
		g.t.Fatalf("concurrencyGuard had %d concurrent callers - only %d permitted", newVal, g.max)
	}
}

func (g *concurrencyGuard) Stop() {
	newVal := g.value.Dec()
	if newVal < 0 {
		g.t.Fatalf("concurrencyGuard callers went below zero - mismatched calls to Start and Stop")
	}
}
