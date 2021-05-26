package parsers

import (
	"testing"

	"github.com/couchbaselabs/cbmultimanager/log/values"
)

func TestIndexCreated(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: "2021-02-19T13:19:54.663+00:00 [Info] clustMgrAgent::OnIndexCreate Success for Create Index DefnId: " +
				"9486123322211990118 Name: def_icao Using: plasma Bucket: travel-sample Scope/Id: _default/0 Collection/Id: " +
				"_default/0 IsPrimary: false NumReplica: 0 InstVersion: 0",
			ExpectedResult: &values.Result{
				Event: values.IndexCreatedEvent,
				Settings: map[string]string{
					"DefnId":       "9486123322211990118",
					"Name":         "def_icao",
					"Using":        "plasma",
					"Bucket":       "travel-sample",
					"ScopeId":      "_default/0",
					"CollectionId": "_default/0",
					"IsPrimary":    "false",
					"NumReplica":   "0",
					"InstVersion":  "0",
				},
			},
		},
		{
			Name: `notInLine`,
			Line: "2021-02-19T13:19:54.663+00:00 [Info] clustMgrAgent::OnIndexCreate Fail for Create Index DefnId:" +
				"9486123322211990118 Name: def_icao Using: plasma Bucket: travel-sample Scope/Id: _default/0 Collection/Id: " +
				"_default/0 IsPrimary: false NumReplica: 0 InstVersion: 0 ",
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, IndexCreated)
}

func TestIndexDeleted(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: "2021-05-17T17:55:00.270+01:00 [Info] clustMgrAgent::OnIndexDelete Success for Drop IndexId " +
				"13770128051184463199",
			ExpectedResult: &values.Result{
				Event: values.IndexDeletedEvent,
				Index: "13770128051184463199",
			},
			ExpectedError: nil,
		},
		{
			Name: `notInLine`,
			Line: "2021-05-17T17:55:00.268+01:00 [Info] clustMgrAgent::OnIndexDelete Notification Received for Drop IndexId" +
				" 13770128051184463199 &{0}",
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, IndexDeleted)
}

func TestIndexerActive(t *testing.T) {
	testCases := []inLineTestCase{
		{
			Name: "inLine",
			Line: "2021-02-19T13:19:37.950+00:00 [Info] Indexer::NewIndexer Status Active",
			ExpectedResult: &values.Result{
				Event: values.IndexerActiveEvent,
			},
		},
		{
			Name:          `notInLine`,
			Line:          "2021-02-19T13:19:37.950+00:00 [Info] Indexer::NewIndexer Status",
			ExpectedError: values.ErrNotInLine,
		},
	}

	runTestCases(t, testCases, IndexerActive)
}
