package utilities

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type lastLineTestCase struct {
	name          string
	input         string
	expectedTime  time.Time
	expectedLines []string
}

func TestGetLastLinesOfEventLog(t *testing.T) {
	timestamp, err := time.Parse(time.RFC3339Nano, "2021-02-19T13:29:37.95Z")
	require.NoError(t, err)

	testCases := []lastLineTestCase{
		{
			name:          "onlyLine",
			input:         "{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\"}\n",
			expectedLines: []string{"{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\"}"},
			expectedTime:  timestamp,
		},
		{
			name: "multiLineWithDifferentTime",
			input: "{\"timestamp\":\"2021-02-19T13:09:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":" +
				"\"2021-02-19T13:19:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":\"2021-02-19T13:29:37.95Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
			expectedLines: []string{"{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\"}"},
			expectedTime:  timestamp,
		},
		{
			name:          "empty",
			input:         "",
			expectedLines: []string{},
			expectedTime:  time.Time{},
		},
		{
			name: "multiLineWithSameTime",
			input: "{\"timestamp\":\"2021-02-19T13:09:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":" +
				"\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_built\"}\n{\"timestamp\":\"2021-02-19T13:29:37.95Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
			expectedLines: []string{
				"{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\"}",
				"{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_built\"}",
			},
			expectedTime: timestamp,
		},
		{
			name: "longLine",
			input: "{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\",\"field\":\"indexer_data\"" +
				",\"field1\":\"indexer_data\",\"field2\":\"indexer_data\",\"field3\":\"indexer_data\",\"field4\":" +
				"\"indexer_data\",\"field5\":\"indexer_data\",\"field6\":\"indexer_data\"}\n",
			expectedLines: []string{
				"{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\",\"field\":\"indexer_data\"" +
					",\"field1\":\"indexer_data\",\"field2\":\"indexer_data\",\"field3\":\"indexer_data\",\"field4\":" +
					"\"indexer_data\",\"field5\":\"indexer_data\",\"field6\":\"indexer_data\"}",
			},
			expectedTime: timestamp,
		},
	}

	for _, x := range testCases {
		t.Run(x.name, func(t *testing.T) {
			logA, err := os.Create("events_test_node.log")
			require.NoError(t, err)
			defer logA.Close()

			_, err = logA.WriteString(x.input)
			require.NoError(t, err)

			timestamp, line, err := GetLastLinesOfEventLog("test_node")
			require.NoError(t, err)
			require.Equal(t, x.expectedTime, timestamp)
			require.Equal(t, x.expectedLines, line)
		})
	}
}
