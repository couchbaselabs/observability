// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package scraper

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/parsers"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"

	"github.com/stretchr/testify/require"
)

type scraperTestCase struct {
	name          string
	result        *values.Result
	continueTime  time.Time
	expectedLine  string
	expectedError error
	lastLine      string
	line          string
	fullLine      string
}

func TestWriteLine(t *testing.T) {
	timestamp, err := time.Parse(time.RFC3339Nano, "2021-02-25T14:50:34.396Z")
	require.NoError(t, err)
	timestampBefore, err := time.Parse(time.RFC3339Nano, "2021-02-24T14:50:34.396Z")
	require.NoError(t, err)

	testCases := []scraperTestCase{
		{
			name: "continueTimeBefore",
			result: &values.Result{
				Time:        timestamp,
				Event:       values.FailoverStartEvent,
				Node:        "ns_1@10.144.210.102",
				OperationID: "8d06a862a8bf0d0a47c083f352b05d29",
			},
			continueTime: timestampBefore,
			expectedLine: "{\"timestamp\":\"2021-02-25T14:50:34.396Z\",\"event_type\":\"failover_start\",\"node\":" +
				"\"ns_1@10.144.210.102\",\"operation_id\":\"8d06a862a8bf0d0a47c083f352b05d29\"}\n",
		},
		{
			name: "continueTimeEqual",
			result: &values.Result{
				Time:        timestamp,
				Event:       values.FailoverStartEvent,
				Node:        "ns_1@10.144.210.102",
				OperationID: "8d06a862a8bf0d0a47c083f352b05d29",
			},
			lastLine: "{\"timestamp\":\"2021-02-25T14:50:34.396Z\",\"event_type\":\"failover_start\",\"node\":" +
				"\"ns_1@10.144.210.102\",\"operation_id\":\"8d06a862a8bf0d0a47c083f352b05d29\"}",
			continueTime:  timestamp,
			expectedError: values.ErrAlreadyInLog,
		},
	}

	for _, x := range testCases {
		t.Run(x.name, func(t *testing.T) {
			dir := t.TempDir()

			logA, err := os.Create(filepath.Join(dir, "test_events.log"))
			require.NoError(t, err)
			defer logA.Close()

			err = writeLine(x.result, x.result.Time, x.continueTime, []string{x.lastLine}, logA)
			require.ErrorIs(t, err, x.expectedError)

			outputFile, err := os.Open(filepath.Join(dir, "test_events.log"))
			require.NoError(t, err)
			defer outputFile.Close()

			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(outputFile)
			require.NoError(t, err)
			require.Equal(t, x.expectedLine, buf.String())
		})
	}
}

func TestRunParsersOnLine(t *testing.T) {
	testCases := []scraperTestCase{
		{
			name: "continueTimeBefore",
			line: "[user:info,2021-02-24T14:50:34.396Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:handle_start_" +
				"failover:1571]Starting failover of nodes ['ns_1@10.144.210.102']. Operation Id = " +
				"8d06a862a8bf0d0a47c083f352b05d29\n",
			fullLine:      "",
			expectedError: values.ErrAlreadyInLog,
		},
		{
			name: "continueTimeAfter",
			line: "[user:info,2021-02-26T14:50:34.396Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:handle_start_" +
				"failover:1571]Starting failover of nodes ['ns_1@10.144.210.102']. Operation Id = " +
				"8d06a862a8bf0d0a47c083f352b05d29\n",
			expectedLine: "{\"timestamp\":\"2021-02-26T14:50:34.396Z\",\"event_type\":\"failover_start\",\"node\":" +
				"\"ns_1@10.144.210.102\",\"operation_id\":\"8d06a862a8bf0d0a47c083f352b05d29\"}\n",
			fullLine: "",
		},
		{
			name: "continuedLine",
			line: "Rebalance Operation Id = f89c0620759d3d341bf3bc0a8e5f03ec\n",
			fullLine: `[user:info,2021-02-25T14:51:11.951Z,ns_1@10.144.210.101:<0.4760.32>:ns_orchestrator:log_rebalance_` +
				`completion:1402]Rebalance completed successfully.`,
			expectedLine: "{\"timestamp\":\"2021-02-25T14:51:11.951Z\",\"event_type\":\"rebalance_finish\",\"successful\":" +
				"true,\"reason\":\"Rebalance completed successfully\",\"operation_id\":\"f89c0620759d3d341bf3bc0a8e5f03ec\"}\n",
		},
	}

	for _, x := range testCases {
		t.Run(x.name, func(t *testing.T) {
			file := parsers.Log{
				Name:           "test",
				StartsWithTime: false,
				Parsers: []parsers.ParserFn{
					parsers.RebalanceFinish,
					parsers.FailoverStartTime,
				},
			}

			dir := t.TempDir()

			logA, err := os.OpenFile(filepath.Join(dir, "test.log"), os.O_CREATE|os.O_RDWR, os.ModePerm)
			require.NoError(t, err)

			_, err = logA.WriteString(x.line)
			require.NoError(t, err)

			logA.Close()

			logC, err := os.Open(filepath.Join(dir, "test.log"))
			require.NoError(t, err)
			defer logC.Close()

			logB, err := os.OpenFile(filepath.Join(dir, "test_events.log"), os.O_CREATE|os.O_RDWR, os.ModePerm)
			require.NoError(t, err)
			defer logB.Close()

			reader := bufio.NewReader(logC)
			require.NoError(t, err)

			continueLine := "{\"timestamp\":\"2021-02-25T14:50:34.396Z\",\"event_type\":\"failover_start\",\"node\":" +
				"\"ns_1@10.144.210.102\",\"operation_id\":\"8d06a862a8bf0d0a47c083f352b05d29\"}"
			continueTime, err := time.Parse(time.RFC3339Nano, "2021-02-25T14:50:34.396Z")
			require.NoError(t, err)

			_, _, err = runParsersOnLine(reader, x.fullLine, continueTime, []string{continueLine}, logB, file, 0)
			require.ErrorIs(t, err, x.expectedError)

			outputFile, err := os.Open(filepath.Join(dir, "test_events.log"))
			require.NoError(t, err)
			defer outputFile.Close()

			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(outputFile)
			require.NoError(t, err)
			require.Equal(t, x.expectedLine, buf.String())
		})
	}
}
