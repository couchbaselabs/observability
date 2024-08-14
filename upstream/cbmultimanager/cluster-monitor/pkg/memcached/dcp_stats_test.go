// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package memcached

import (
	"testing"

	memcached "github.com/couchbase/gomemcached/client"
	"github.com/stretchr/testify/require"
)

func TestParseStatValue(t *testing.T) {
	cases := []struct {
		name     string
		input    []memcached.StatValue
		expected *DCPMemStats
	}{
		{
			name: "unknown",
			input: []memcached.StatValue{
				{
					Key: `eq_dcpq:nonexistent:asdasd:some_stat`,
					Val: "0",
				},
			},
			expected: &DCPMemStats{
				StreamNames: []string{
					"nonexistent:asdasd",
				},
			},
		},
		{
			name: "replication/max_buffer_bytes",
			input: []memcached.StatValue{
				{
					Key: "eq_dcpq:replication:ns_1@127.0.0.1->ns_2@127.0.0.1:bucket:max_buffer_bytes",
					Val: "100",
				},
			},
			expected: &DCPMemStats{
				MaxBufferBytes: []ReplicationStat{
					{
						Source:      "ns_1@127.0.0.1",
						Destination: "ns_2@127.0.0.1",
						Name:        "max_buffer_bytes",
						Value:       "100",
						Extras:      "",
					},
				},
				StreamNames: []string{
					`replication:ns_1@127.0.0.1->ns_2@127.0.0.1:bucket`,
				},
			},
		},
		{
			name: "replication/unacked_bytes",
			input: []memcached.StatValue{
				{
					Key: "eq_dcpq:replication:ns_1@127.0.0.1->ns_2@127.0.0.1:bucket:unacked_bytes",
					Val: "100",
				},
			},
			expected: &DCPMemStats{
				UnackedBytes: []ReplicationStat{
					{
						Source:      "ns_1@127.0.0.1",
						Destination: "ns_2@127.0.0.1",
						Name:        "unacked_bytes",
						Value:       "100",
						Extras:      "",
					},
				},
				StreamNames: []string{
					`replication:ns_1@127.0.0.1->ns_2@127.0.0.1:bucket`,
				},
			},
		},
		{
			name: "secidx",
			input: []memcached.StatValue{
				{
					Key: "eq_dcpq:secidx:proj-travel-sample-MAINT_STREAM_TOPIC_83cca6463f58fe78e740f4c6a8c2936e" +
						"-18380406057854148987/0:max_buffer_bytes",
					Val: "100",
				},
			},
			// shouldn't result in a ReplicationStat since there's no source and destination
			expected: &DCPMemStats{
				StreamNames: []string{
					"secidx:proj-travel-sample-MAINT_STREAM_TOPIC_83cca6463f58fe78e740f4c6a8c2936e" +
						"-18380406057854148987/0",
				},
			},
		},
		{
			name: "general",
			input: []memcached.StatValue{
				{
					Key: "ep_dcp_count",
					Val: "5",
				},
			},
			expected: &DCPMemStats{
				Count: "5",
			},
		},
	}

	for i := range cases {
		t.Run(cases[i].name, func(t *testing.T) {
			tc := cases[i]
			result := parseDCPStatValues(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
