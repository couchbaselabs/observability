// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package configuration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLabelSelector(t *testing.T) {
	t.Run("OneOK", func(t *testing.T) {
		t.Parallel()
		result, err := ParseLabelSelectors("key=value")
		require.NoError(t, err)
		require.Equal(t, LabelSelectors{"key": "value"}, result)
	})
	t.Run("MultiOK", func(t *testing.T) {
		t.Parallel()
		result, err := ParseLabelSelectors("key1=value1 key2=value2")
		require.NoError(t, err)
		require.Equal(t, LabelSelectors{"key1": "value1", "key2": "value2"}, result)
	})
	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		result, err := ParseLabelSelectors("")
		require.NoError(t, err)
		require.Equal(t, LabelSelectors{}, result)
	})
	t.Run("Invalid", func(t *testing.T) {
		t.Parallel()
		_, err := ParseLabelSelectors("total nonsense")
		require.Error(t, err)
	})
}
