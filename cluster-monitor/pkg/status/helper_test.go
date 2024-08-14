// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package status

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/couchbase/tools-common/cbvalue"
	"github.com/stretchr/testify/require"
)

func TestVersionCheck(t *testing.T) {
	defs := map[string]values.CheckerDefinition{
		"a": {
			MinVersion: cbvalue.Version7_0_0,
		},
		"b": {
			MinVersion: cbvalue.Version6_0_0,
		},
	}

	t.Run("equal-to-min", func(t *testing.T) {
		require.Nil(t, versionCheck("a", &values.CouchbaseCluster{
			NodesSummary: values.NodesSummary{{Version: string(cbvalue.Version7_0_0)}},
		}, defs))
	})

	t.Run("larger-than-min", func(t *testing.T) {
		require.Nil(t, versionCheck("b", &values.CouchbaseCluster{
			NodesSummary: values.NodesSummary{{Version: string(cbvalue.Version7_0_0)}},
		}, defs))
	})

	t.Run("smaller-than-min", func(t *testing.T) {
		require.NotNil(t, versionCheck("a", &values.CouchbaseCluster{
			NodesSummary: values.NodesSummary{{Version: string(cbvalue.Version6_0_0)}},
		}, defs))
	})

	t.Run("dev-cluster", func(t *testing.T) {
		require.Nil(t, versionCheck("a", &values.CouchbaseCluster{
			NodesSummary: values.NodesSummary{{Version: string(cbvalue.VersionUnknown)}},
		}, defs))
	})
}
