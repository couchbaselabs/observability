// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package couchbase

import (
	"net/http"
	"testing"

	"github.com/couchbase/tools-common/cbrest"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"

	"github.com/stretchr/testify/require"
)

func TestGetAutoFailOverSettings(t *testing.T) {
	var (
		statusCode int
		settings   values.AutoFailoverSettings
	)

	handlers := make(cbrest.TestHandlers)
	handlers.Add(http.MethodGet, string(AutoFailOverSettings), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(statusCode, &settings, []byte{}, w)
	})

	cluster := cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: true,
		UUID:       "cluster_0",
		Nodes:      cbrest.TestNodes{&cbrest.TestNode{}},
		Handlers:   handlers,
	})
	defer cluster.Close()

	client := getTestClient(t, cluster.URL())

	type testCase struct {
		name       string
		returnCode int
		settings   values.AutoFailoverSettings
	}

	cases := []testCase{
		{
			name:       "401",
			returnCode: http.StatusUnauthorized,
			settings:   values.AutoFailoverSettings{Enabled: true},
		},
		{
			name:       "enabled",
			returnCode: http.StatusOK,
			settings:   values.AutoFailoverSettings{Enabled: true},
		},
		{
			name:       "disable",
			returnCode: http.StatusOK,
			settings:   values.AutoFailoverSettings{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			statusCode = tc.returnCode
			settings = tc.settings

			settingsOut, err := client.GetAutoFailOverSettings()
			if tc.returnCode == http.StatusOK {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			if tc.returnCode != http.StatusOK {
				return
			}

			require.Equal(t, &settings, settingsOut)
		})
	}
}
