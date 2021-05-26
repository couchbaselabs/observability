package scraper

import (
	"bytes"
	"os"
	"testing"

	"github.com/couchbaselabs/cbmultimanager/log/values"

	"github.com/stretchr/testify/require"
)

type filterTestCase struct {
	name           string
	input          string
	expectedOutput string
	includedEvents []values.EventType
	excludedEvents []values.EventType
}

func TestFilterIncludeEvents(t *testing.T) {
	testCases := []filterTestCase{
		{
			name: "allIncluded",
			input: "{\"timestamp\":\"2021-03-03T15:50:43.216Z\",\"event_type\":\"password_policy_changed\"}\n" +
				"{\"timestamp\":\"2021-03-03T15:54:27.176Z\",\"event_type\":\"node_went_down\",\"node\":\"ns_1@cb.local\"," +
				"\"reason\":\"[{nodedown_reason,net_kernel_terminated}]\"}\n{\"timestamp\":\"2021-03-03T15:54:28.626Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
			expectedOutput: "{\"timestamp\":\"2021-03-03T15:50:43.216Z\",\"event_type\":\"password_policy_changed\"}\n" +
				"{\"timestamp\":\"2021-03-03T15:54:27.176Z\",\"event_type\":\"node_went_down\",\"node\":\"ns_1@cb.local\"," +
				"\"reason\":\"[{nodedown_reason,net_kernel_terminated}]\"}\n{\"timestamp\":\"2021-03-03T15:54:28.626Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
			includedEvents: []values.EventType{
				values.PasswordPolicyChangedEvent,
				values.NodeWentDownEvent,
				values.IndexerActiveEvent,
			},
		},
		{
			name: "noneIncluded",
			input: "{\"timestamp\":\"2021-03-03T15:50:43.216Z\",\"event_type\":\"password_policy_changed\"}\n" +
				"{\"timestamp\":\"2021-03-03T15:54:27.176Z\",\"event_type\":\"node_went_down\",\"node\":\"ns_1@cb.local\"," +
				"\"reason\":\"[{nodedown_reason,net_kernel_terminated}]\"}\n{\"timestamp\":\"2021-03-03T15:54:28.626Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
			expectedOutput: "",
			includedEvents: []values.EventType{
				values.GroupAddedEvent,
			},
		},
		{
			name: "someIncluded",
			input: "{\"timestamp\":\"2021-03-03T15:50:43.216Z\",\"event_type\":\"password_policy_changed\"}\n" +
				"{\"timestamp\":\"2021-03-03T15:54:27.176Z\",\"event_type\":\"node_went_down\",\"node\":\"ns_1@cb.local\"," +
				"\"reason\":\"[{nodedown_reason,net_kernel_terminated}]\"}\n{\"timestamp\":\"2021-03-03T15:54:28.626Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
			expectedOutput: "{\"timestamp\":\"2021-03-03T15:50:43.216Z\",\"event_type\":\"password_policy_changed\"}\n" +
				"{\"timestamp\":\"2021-03-03T15:54:28.626Z\",\"event_type\":\"indexer_active\"}\n",
			includedEvents: []values.EventType{
				values.PasswordPolicyChangedEvent,
				values.IndexerActiveEvent,
			},
		},
		{
			name: "allExcluded",
			input: "{\"timestamp\":\"2021-03-03T15:50:43.216Z\",\"event_type\":\"password_policy_changed\"}\n" +
				"{\"timestamp\":\"2021-03-03T15:54:27.176Z\",\"event_type\":\"node_went_down\",\"node\":\"ns_1@cb.local\"," +
				"\"reason\":\"[{nodedown_reason,net_kernel_terminated}]\"}\n{\"timestamp\":\"2021-03-03T15:54:28.626Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
			expectedOutput: "{\"timestamp\":\"2021-03-03T15:50:43.216Z\",\"event_type\":\"password_policy_changed\"}\n" +
				"{\"timestamp\":\"2021-03-03T15:54:27.176Z\",\"event_type\":\"node_went_down\",\"node\":\"ns_1@cb.local\"," +
				"\"reason\":\"[{nodedown_reason,net_kernel_terminated}]\"}\n{\"timestamp\":\"2021-03-03T15:54:28.626Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
			excludedEvents: []values.EventType{
				values.GroupAddedEvent,
			},
		},
		{
			name: "noneExcluded",
			input: "{\"timestamp\":\"2021-03-03T15:50:43.216Z\",\"event_type\":\"password_policy_changed\"}\n" +
				"{\"timestamp\":\"2021-03-03T15:54:27.176Z\",\"event_type\":\"node_went_down\",\"node\":\"ns_1@cb.local\"," +
				"\"reason\":\"[{nodedown_reason,net_kernel_terminated}]\"}\n{\"timestamp\":\"2021-03-03T15:54:28.626Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
			expectedOutput: "",
			excludedEvents: []values.EventType{
				values.PasswordPolicyChangedEvent,
				values.NodeWentDownEvent,
				values.IndexerActiveEvent,
			},
		},
		{
			name: "someExcluded",
			input: "{\"timestamp\":\"2021-03-03T15:50:43.216Z\",\"event_type\":\"password_policy_changed\"}\n" +
				"{\"timestamp\":\"2021-03-03T15:54:27.176Z\",\"event_type\":\"node_went_down\",\"node\":\"ns_1@cb.local\"," +
				"\"reason\":\"[{nodedown_reason,net_kernel_terminated}]\"}\n{\"timestamp\":\"2021-03-03T15:54:28.626Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
			expectedOutput: "{\"timestamp\":\"2021-03-03T15:50:43.216Z\",\"event_type\":\"password_policy_changed\"}\n" +
				"{\"timestamp\":\"2021-03-03T15:54:28.626Z\",\"event_type\":\"indexer_active\"}\n",
			excludedEvents: []values.EventType{
				values.NodeWentDownEvent,
			},
		},
	}

	for _, x := range testCases {
		t.Run(x.name, func(t *testing.T) {
			os.Remove("filtered_events_test_node.log")

			cred := &values.Credentials{
				User:     "Administrator",
				Password: "password",
				Cluster:  "http://example:9000/",
				NodeName: "test_node",
			}

			logA, err := os.Create("events_test_node.log")
			require.NoError(t, err)
			defer logA.Close()

			_, err = logA.WriteString(x.input)
			require.NoError(t, err)

			if x.includedEvents != nil {
				err = FilterEvents(cred, x.includedEvents, true)
			} else {
				err = FilterEvents(cred, x.excludedEvents, false)
			}

			require.NoError(t, err)

			outputFile, err := os.Open("filtered_events_test_node.log")
			require.NoError(t, err)
			defer outputFile.Close()

			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(outputFile)
			require.NoError(t, err)
			require.Equal(t, x.expectedOutput, buf.String())
		})
	}
}
