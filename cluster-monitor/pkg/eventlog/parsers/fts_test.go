// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package parsers

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"
)

func TestFTSIndexCreatedOrDropped(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLineftsIndexCreated",
			Line: `2021-03-04T14:14:58.600+00:00 [INFO] manager_api: index definition created, indexType: fulltext-index,` +
				`indexName: test-fts-index, indexUUID: 7db45cdd54cd4630`,
			ExpectedResult: &values.Result{
				Event: values.FTSIndexCreatedEvent,
				Index: "test-fts-index",
			},
		},
		{
			Name: "inLineftsIndexDropped",
			Line: `2021-03-04T14:15:20.507+00:00 [INFO] manager_api: index definition deleted, indexType: fulltext-index,` +
				`indexName: gvbvbn, indexUUID: 416bd4854a8f2c01`,
			ExpectedResult: &values.Result{
				Event: values.FTSIndexDroppedEvent,
				Index: "gvbvbn",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-03-04T14:15:20.507+00:00 [INFO] manager_api: index definition dropped, indexType: ` +
				`fulltext-index, indexName: gvbvbn, indexUUID: 416bd4854a8f2c01`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, FTSIndexCreatedOrDropped)
}
