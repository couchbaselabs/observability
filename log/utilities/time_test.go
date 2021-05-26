package utilities

import (
	"testing"
	"time"

	"github.com/couchbaselabs/cbmultimanager/log/values"

	"github.com/stretchr/testify/require"
)

type parseTimeTestCase struct {
	name          string
	input         string
	expectedTime  time.Time
	errorExpected bool
}

func TestGetTime(t *testing.T) {
	timestampSame, err := time.Parse(time.RFC3339Nano, "2021-02-19T13:29:37.95Z")
	require.NoError(t, err)
	timestampDifferent, err := time.Parse(time.RFC3339Nano, "2021-02-19T14:29:37.95+01:00")
	require.NoError(t, err)

	testCases := []parseTimeTestCase{
		{
			name:         "sameTimeZone",
			input:        "{\"timestamp\":\"2021-02-19T13:29:37.95Z\",\"event_type\":\"indexer_active\"}\n",
			expectedTime: timestampSame,
		},
		{
			name:         "differntTimeZone",
			input:        "{\"timestamp\":\"2021-02-19T14:29:37.95+01:00\",\"event_type\":\"indexer_active\"}\n",
			expectedTime: timestampDifferent,
		},
		{
			name:          "notInLine",
			input:         "{\"timestamp\":2021-02-19T14:29:37.95+01:00,\"event_type\":\"indexer_active\"}\n",
			errorExpected: true,
		},
	}

	for _, x := range testCases {
		t.Run(x.name, func(t *testing.T) {
			timestamp, err := GetTime(x.input)
			if x.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, x.expectedTime, timestamp)
			}
		})
	}
}

type stringTimeTestCase struct {
	name           string
	line           string
	logName        string
	expectedTime   time.Time
	expectedError  error
	startsWithTime bool
}

func TestGetTimeFromString(t *testing.T) {
	timestamp, err := time.Parse(time.RFC3339Nano, "2021-02-19T13:29:37.95Z")
	require.NoError(t, err)

	testCases := []stringTimeTestCase{
		{
			name: "backupLine",
			line: `2021-02-19T13:29:37.95Z	INFO	(Event Handler) (Node Run) Task finished  {"cluster": "self", ` +
				`"repository": "simple", "task": "BACKUP-142a8e59-c3d3-4082-8195-6529e9b15a16", "status": "done"}`,
			logName:        "backup_service",
			startsWithTime: true,
			expectedTime:   timestamp,
		},
		{
			name: "diagLine",
			line: `2021-02-19T13:29:37.95Z, ns_cluster:3:info:message(ns_1@10.144.210.102) - Node ns_1@10.144.210.102 ` +
				`joined cluster`,
			logName:        "diag",
			startsWithTime: true,
			expectedTime:   timestamp,
		},
		{
			name: "infoLine",
			line: `[menelaus:info,2021-02-19T13:29:37.95Z,ns_1@10.144.210.101:<0.27363.671>:menelaus_web_buckets:` +
				`handle_bucket_delete:408]Deleted bucket "beer-sample"`,
			logName:      "info",
			expectedTime: timestamp,
		},
		{
			name:           "indexerLine",
			line:           "2021-02-19T13:29:37.95Z [Info] Indexer::NewIndexer Status Active",
			logName:        "indexer",
			startsWithTime: true,
			expectedTime:   timestamp,
		},
		{
			name: "goxdcrLine",
			line: `2021-02-19T13:29:37.95Z ERRO GOXDCR.AdminPort: Error in replication. errorsMap=map[toBucket:Error` +
				`validating target bucket 'target'. err=BucketValidationInfo Operation failed after max retries.  Last error: ` +
				`Bucket doesn't exist]`,
			logName:        "goxdcr",
			startsWithTime: true,
			expectedTime:   timestamp,
		},
		{
			name: "ftsLine",
			line: `2021-02-19T13:29:37.95Z [INFO] manager_api: index definition deleted, indexType: fulltext-index,` +
				`indexName: gvbvbn, indexUUID: 416bd4854a8f2c01`,
			logName:        "fts",
			startsWithTime: true,
			expectedTime:   timestamp,
		},
		{
			name: "debugLine",
			line: `[ns_server:debug,2021-02-19T13:29:37.95Z,ns_1@10.144.210.101:ns_config_log<0.221.0>:ns_config_log:` +
				`log_common:232]config change:password_policy ->[{min_length,6},{must_present,[]}]`,
			logName:      "debug",
			expectedTime: timestamp,
		},
		{
			name: "incorrectStartsWith",
			line: `[ns_server:debug,2021-02-19T13:29:37.95Z,ns_1@10.144.210.101:ns_config_log<0.221.0>:ns_config_log:` +
				`log_common:232]config change:password_policy ->[{min_length,6},{must_present,[]}]`,
			logName:        "debug",
			startsWithTime: true,
			expectedError:  values.ErrNotInLine,
		},
		{
			name:          "incorrectLine",
			line:          `This is an example line`,
			logName:       "debug",
			expectedError: values.ErrNotInLine,
		},
	}

	for _, x := range testCases {
		t.Run(x.name, func(t *testing.T) {
			timestamp, err := GetTimeFromString(x.startsWithTime, x.line, x.logName)
			require.ErrorIs(t, x.expectedError, err)
			require.Equal(t, x.expectedTime, timestamp)
		})
	}
}
