// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package scraper

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/parsers"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"

	"github.com/stretchr/testify/require"
)

type mergeTestCase struct {
	name           string
	input1         string
	input2         string
	input3         string
	expectedOutput string
	expectedError  error
}

func TestMerge(t *testing.T) {
	testCases := []mergeTestCase{
		{
			name:   "outOfOrder",
			input1: "{\"timestamp\":\"2021-02-19T13:19:37.95Z\",\"event_type\":\"indexer_active\"}\n",
			input2: "{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\"}\n",
			input3: "{\"timestamp\":\"2021-02-19T13:09:37.95Z\",\"event_type\":\"indexer_active\"}\n",
			expectedOutput: "{\"timestamp\":\"2021-02-19T13:09:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":" +
				"\"2021-02-19T13:19:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":\"2021-02-19T13:29:37.95Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
		},
		{
			name:   "inOrder",
			input1: "{\"timestamp\":\"2021-02-19T13:19:37.95Z\",\"event_type\":\"indexer_active\"}\n",
			input2: "{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\"}\n",
			input3: "{\"timestamp\":\"2021-02-19T13:39:37.95Z\",\"event_type\":\"indexer_active\"}\n",
			expectedOutput: "{\"timestamp\":\"2021-02-19T13:19:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":" +
				"\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":\"2021-02-19T13:39:37.95Z\"," +
				"\"event_type\":\"indexer_active\"}\n",
		},
		{
			name: "multi-line",
			input1: "{\"timestamp\":\"2021-02-19T13:19:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":" +
				"\"2021-02-19T13:49:37.95Z\",\"event_type\":\"indexer_active\"}\n",
			input2: "{\"timestamp\":\"2021-02-19T13:09:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":" +
				"\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\"}\n",
			input3: "{\"timestamp\":\"2021-02-19T13:39:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":" +
				"\"2021-02-19T13:59:37.95Z\",\"event_type\":\"indexer_active\"}\n",
			expectedOutput: "{\"timestamp\":\"2021-02-19T13:09:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":" +
				"\"2021-02-19T13:19:37.95Z\",\"event_type\":\"indexer_active\"}\n{\"timestamp\":\"2021-02-19T13:29:37.95Z\"," +
				"\"event_type\":\"indexer_active\"}\n{\"timestamp\":\"2021-02-19T13:39:37.95Z\",\"event_type\":" +
				"\"indexer_active\"}\n{\"timestamp\":\"2021-02-19T13:49:37.95Z\",\"event_type\":\"indexer_active\"}\n" +
				"{\"timestamp\":\"2021-02-19T13:59:37.95Z\",\"event_type\":\"indexer_active\"}\n",
		},
		{
			name:           "empty",
			input1:         ``,
			input2:         ``,
			input3:         ``,
			expectedOutput: ``,
		},
	}

	for _, x := range testCases {
		t.Run(x.name, func(t *testing.T) {
			dir := t.TempDir()

			cred := &values.Credentials{
				User:     "Administrator",
				Password: "password",
				Cluster:  "http://example:9000/",
				NodeName: "test_node",
			}

			parsers.ParserFunctions = []parsers.Log{
				{
					Name:           "logA",
					StartsWithTime: false,
				},
				{
					Name:           "logB",
					StartsWithTime: true,
				},
				{
					Name:           "logC",
					StartsWithTime: true,
				},
			}

			logA, err := os.Create(filepath.Join(dir, "logA_events.log"))
			require.NoError(t, err)
			defer logA.Close()

			logB, err := os.Create(filepath.Join(dir, "logB_events.log"))
			require.NoError(t, err)
			defer logB.Close()

			logC, err := os.Create(filepath.Join(dir, "logC_events.log"))
			require.NoError(t, err)
			defer logC.Close()

			_, err = logA.WriteString(x.input1)
			require.NoError(t, err)
			_, err = logB.WriteString(x.input2)
			require.NoError(t, err)
			_, err = logC.WriteString(x.input3)
			require.NoError(t, err)

			err = MergeEventLogs(cred, dir)
			require.ErrorIs(t, err, x.expectedError)

			outputFile, err := os.Open(filepath.Join(dir, "events_test_node.log"))
			require.NoError(t, err)
			defer outputFile.Close()

			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(outputFile)
			require.NoError(t, err)
			require.Equal(t, x.expectedOutput, buf.String())
		})
	}
}
