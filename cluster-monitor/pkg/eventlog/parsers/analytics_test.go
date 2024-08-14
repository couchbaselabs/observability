// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package parsers

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"
)

func TestAnalyticsCollectionCreated(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `2021-05-20T10:11:39.636+01:00, analytics:0:info:message(n_3@127.0.0.1) - Created analytics collection ` +
				`Default.bee on data service collection beer-sample._default._default`,
			ExpectedResult: &values.Result{
				Event:               values.AnalyticsCollectionCreatedEvent,
				Collection:          "beer-sample._default._default",
				AnalyticsCollection: "Default.bee",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-02-19T03:19:26.330-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Created data ` +
				`"dv_1".`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, AnalyticsCollectionCreated)
}

func TestAnalyticsCollectionDropped(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `2021-05-20T10:11:46.666+01:00, analytics:0:info:message(n_3@127.0.0.1) - Dropped analytics collection ` +
				`Default.bee`,
			ExpectedResult: &values.Result{
				Event:               values.AnalyticsCollectionDroppedEvent,
				AnalyticsCollection: "Default.bee",
			},
		},
		{
			Name: "notInLine",
			Line: `2020-06-22T14:34:21.533Z, analytics:0:info:message(ns_1@10.17.124.92) - Dropped data ` +
				`"crew_effects".`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, AnalyticsCollectionDropped)
}

func TestAnalyticsIndexCreatedOrDropped(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "indexCreatedOnDatatset",
			Line: `2021-02-19T03:24:37.959-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Created index "idx_1"` +
				` on shadow dataset "ds_3".`,
			ExpectedResult: &values.Result{
				Event:   values.AnalyticsIndexCreatedEvent,
				Dataset: "\"ds_3\".",
				Index:   "\"idx_1\"",
			},
		},
		{
			Name: "indexDroppedFromDatatset",
			Line: `2021-02-19T14:08:22.824Z, analytics:0:info:message(n_0@cb.local) - Dropped index "analytics_index" ` +
				`from shadow dataset "analytics_data".`,
			ExpectedResult: &values.Result{
				Event:   values.AnalyticsIndexDroppedEvent,
				Dataset: "\"analytics_data\".",
				Index:   "\"analytics_index\"",
			},
		},
		{
			Name: "indexCreatedOnCollection",
			Line: `2021-02-19T03:24:37.959-08:00, analytics:0:info:message(n_1@127.0.0.1) - Created ` +
				`index brewerIndex on analytics collection Default.test`,
			ExpectedResult: &values.Result{
				Event:               values.AnalyticsIndexCreatedEvent,
				AnalyticsCollection: "Default.test",
				Index:               "brewerIndex",
			},
		},
		{
			Name: "indexDroppedFromCollection",
			Line: `2021-02-19T14:08:22.824Z, analytics:0:info:message(n_3@127.0.0.1) - Dropped index ` +
				`beers_name_idx from analytics collection Default.beers`,
			ExpectedResult: &values.Result{
				Event:               values.AnalyticsIndexDroppedEvent,
				AnalyticsCollection: "Default.beers",
				Index:               "beers_name_idx",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-02-19T03:24:37.959-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Create index "idx_1"` +
				` on shadow dataset "ds_3".`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, AnalyticsIndexCreatedOrDropped)
}

func TestAnalyticsScopeCreatedOrDropped(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLineCreated",
			Line: `2021-05-20T10:26:18.008+01:00, analytics:0:info:message(n_3@127.0.0.1) - Created analytics scope bee`,
			ExpectedResult: &values.Result{
				Event:          values.AnalyticsScopeCreatedEvent,
				AnalyticsScope: "bee",
			},
		},
		{
			Name: "inLineDropped",
			Line: `2021-05-20T10:26:27.577+01:00, analytics:0:info:message(n_3@127.0.0.1) - Dropped analytics scope bee`,
			ExpectedResult: &values.Result{
				Event:          values.AnalyticsScopeDroppedEvent,
				AnalyticsScope: "bee",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-02-19T03:19:26.330-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Created data ` +
				`"dv_1".`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, AnalyticsScopeCreatedOrDropped)
}

func TestLinkConnectedOrDisconnected(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLineConnected",
			Line: `2021-05-20T09:40:49.143+01:00, analytics:0:info:message(n_3@127.0.0.1) - Connected bucket beer-sample` +
				` for link Default.Local`,
			ExpectedResult: &values.Result{
				Event: values.AnalyticsLinkConnectedEvent,
				Link:  "Default.Local",
			},
		},
		{
			Name: "inLineDisconnected",
			Line: `2021-05-13T11:30:35.579+01:00, analytics:0:info:message(n_1@127.0.0.1) - Disconnected link Default.Local`,
			ExpectedResult: &values.Result{
				Event: values.AnalyticsLinkDisconnectedEvent,
				Link:  "Default.Local",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-03-12T09:02:12.756-08:00, analytics:0:info:message(ns_1@172.23.105.20) - Created link ` +
				`"Default.Local".`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, LinkConnectedOrDisconnected)
}

func TestDataverseCreatedOrDropped(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `2021-02-19T03:19:26.330-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Created dataverse ` +
				`"dv_1".`,
			ExpectedResult: &values.Result{
				Event:     values.DataverseCreatedEvent,
				Dataverse: "dv_1",
			},
		},
		{
			Name: "inLine",
			Line: `2021-02-19T03:20:02.871-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Dropped dataverse ` +
				`"dv_1".`,
			ExpectedResult: &values.Result{
				Event:     values.DataverseDroppedEvent,
				Dataverse: "dv_1",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-02-19T03:19:26.330-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Created data ` +
				`"dv_1".`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, DataverseCreatedOrDropped)
}

func TestDatasetCreated(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `2021-02-19T03:19:33.248-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Created dataset "ds_1"` +
				` on bucket "bucket6".`,
			ExpectedResult: &values.Result{
				Event:   values.DatasetCreatedEvent,
				Bucket:  "bucket6",
				Dataset: "ds_1",
			},
		},
		{
			Name: "notInLine",
			Line: `2021-02-19T03:19:26.330-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Created data ` +
				`"dv_1".`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, DatasetCreated)
}

func TestDatasetDropped(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: `2020-06-22T14:34:21.533Z, analytics:0:info:message(ns_1@10.17.124.92) - Dropped dataset "crew_effects".`,
			ExpectedResult: &values.Result{
				Event:   values.DatasetDroppedEvent,
				Dataset: "crew_effects",
			},
		},
		{
			Name: "notInLine",
			Line: `2020-06-22T14:34:21.533Z, analytics:0:info:message(ns_1@10.17.124.92) - Dropped data ` +
				`"crew_effects".`,
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, DatasetDropped)
}
