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

func testMapToSlice(len int, in map[int]map[string]string) BucketCheckpointStats {
	result := make([]map[string]string, len)
	for k, v := range in {
		result[k] = v
	}
	return result
}

func TestParseCheckpointStats(t *testing.T) {
	testCases := []struct {
		name     string
		stats    []memcached.StatValue
		expected []map[string]string
		err      bool
	}{
		{
			name:     "Empty",
			stats:    []memcached.StatValue{},
			expected: BucketCheckpointStats{nil},
		},
		{
			name:     "Simple",
			stats:    []memcached.StatValue{{Key: "vb_0:test", Val: "test"}},
			expected: BucketCheckpointStats{{"test": "test"}},
		},
		{
			name: "MultiVB",
			stats: []memcached.StatValue{
				{Key: "vb_0:test", Val: "test"},
				{Key: "vb_100:test", Val: "test"},
				{Key: "vb_1000:test", Val: "test"},
			},
			expected: testMapToSlice(1024, map[int]map[string]string{
				0:    {"test": "test"},
				100:  {"test": "test"},
				1000: {"test": "test"},
			}),
		},
		{
			name:     "Invalid",
			stats:    []memcached.StatValue{{Key: "XXX", Val: "invalid"}},
			expected: nil,
			err:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseCheckpointStats(tc.stats)

			if !tc.err {
				require.NoError(t, err)
				for vb, actual := range result {
					require.EqualValuesf(t, tc.expected[vb], actual, "vBucket %d", vb)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}
