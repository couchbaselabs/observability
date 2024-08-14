// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package store

import (
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStore(t *testing.T) {
	store := NewInMemoryStore()

	checkers := store.GetCheckers()
	require.Len(t, checkers, 0)

	res := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   "fake-checker",
			Status: values.MissingCheckerStatus,
			Time:   time.Now().UTC(),
		},
		Error: assert.AnError,
	}

	store.SetCheckerResult("fake-checker", res)

	checkers = store.GetCheckers()
	require.Equal(t, map[string]*values.WrappedCheckerResult{"fake-checker": res}, checkers)

	checker, err := store.GetCheckerResult("fake-checker")
	require.NoError(t, err)
	require.Equal(t, res, checker)

	_, err = store.GetCheckerResult("checker-2")
	require.ErrorIs(t, err, values.ErrNotFound)
}
