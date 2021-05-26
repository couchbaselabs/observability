package couchbase

import (
	"net/http"
	"testing"

	"github.com/couchbase/tools-common/cbrest"

	"github.com/stretchr/testify/require"
)

func TestGetAutoFailOverSettings(t *testing.T) {
	var (
		statusCode int
		settings   AutoFailoverSettings
	)

	handlers := make(cbrest.TestHandlers)
	handlers.Add(http.MethodGet, string(AutoFailOverSettings), func(w http.ResponseWriter, r *http.Request) {
		marshalAndSendTestHelper(statusCode, &settings, []byte{}, w)
	})

	cluster := cbrest.NewTestCluster(t, cbrest.TestClusterOptions{
		Enterprise: true,
		UUID:       "cluster_0",
		Nodes:      cbrest.TestNodes{{}},
		Handlers:   handlers,
	})
	defer cluster.Close()

	client := getTestClient(t, cluster.URL())

	type testCase struct {
		name       string
		returnCode int
		settings   AutoFailoverSettings
	}

	cases := []testCase{
		{
			name:       "401",
			returnCode: http.StatusUnauthorized,
			settings:   AutoFailoverSettings{Enabled: true},
		},
		{
			name:       "enabled",
			returnCode: http.StatusOK,
			settings:   AutoFailoverSettings{Enabled: true},
		},
		{
			name:       "disable",
			returnCode: http.StatusOK,
			settings:   AutoFailoverSettings{},
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
