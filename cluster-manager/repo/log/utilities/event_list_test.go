package utilities

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/log/values"

	"github.com/stretchr/testify/require"
)

type eventListTestCase struct {
	name           string
	input          string
	expectedOutput []values.EventType
	expectedError  string
}

func TestGetEventList(t *testing.T) {
	testCases := []eventListTestCase{
		{
			name:           "correctEventString",
			input:          "node_went_down",
			expectedOutput: []values.EventType{values.NodeWentDownEvent},
		},
		{
			name:          "incorrectEventString",
			input:         "node_down",
			expectedError: "invalid event type given: node_down",
		},
		{
			name:           "correctMultipleEventString",
			input:          "node_went_down,password_policy_changed",
			expectedOutput: []values.EventType{values.NodeWentDownEvent, values.PasswordPolicyChangedEvent},
		},
		{
			name:          "containsIncorrectEventString",
			input:         "node_went_down,password_changed",
			expectedError: "invalid event type given: password_changed",
		},
		{
			name:  "emptyEventString",
			input: "",
		},
		{
			name:           "repeatedEventString",
			input:          "node_went_down,node_went_down",
			expectedOutput: []values.EventType{values.NodeWentDownEvent},
		},
	}

	for _, x := range testCases {
		t.Run(x.name, func(t *testing.T) {
			eventList, err := GetEventList(x.input)
			require.Equal(t, x.expectedOutput, eventList)
			if x.expectedError != "" {
				require.EqualError(t, err, x.expectedError)
			}
		})
	}
}
