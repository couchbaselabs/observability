// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package parsers

import (
	"regexp"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"
)

var (
	// capture the analytics and dataservice collections
	// for example from line: "Created analytics collection Default.bee on data service collection
	//	beer-sample._default._default" it would capture "Default.bee" and "beer-sample._default._default"
	collectionsRegexp = regexp.MustCompile(`analytics\scollection\s(?P<collA>[^\s]*)\s.*data\sservice\scollection\s` +
		`(?P<collD>.*)`)
	// capture just the analytics collection
	// for example from line: "Dropped analytics collection Default.bee" it would capture "Default.bee"
	collectionRegexp = regexp.MustCompile(`analytics\scollection\s(?P<collA>.*)`)
	// capture the index name and whether its a collection or dataset along with the collection or dataset name
	// for example from line: "Created index brewerIndex on analytics collection Default.test" it would capture
	//	"brewerIndex", "collection" and "Default.test"
	indexRegexp = regexp.MustCompile(`index\s(?P<index>[^\s]*)\s.*(?P<field>(?:collection|dataset))\s` +
		`(?P<name>.*)`)
	// capture the analytics scope
	scopeRegexp = regexp.MustCompile(`analytics\sscope\s(?P<scopeA>.*)`)
	// capture the link
	linkRegexp = regexp.MustCompile(`link\s(?P<link>.*)`)
	// capture the name of the dataverse
	dataverseRegexp = regexp.MustCompile(`dataverse\s"(?P<dataverse>[^"]*)"`)
	// capture the dataset along with the bucket
	datasetCreatedRegexp = regexp.MustCompile(`dataset\s"(?P<datatset>[^"]*)".*bucket\s"(?P<bucket>[^"]*)"`)
	// capture just the dataset
	datasetDroppedRegexp = regexp.MustCompile(`dataset\s"(?P<datatset>[^"]*)"`)
)

// AnalyticsCollectionCreated gets when an analytics collection is created.
// Example line: 2021-05-20T10:11:39.636+01:00, analytics:0:info:message(n_3@127.0.0.1) - Created analytics collection
//
//	Default.bee on data service collection beer-sample._default._default
func AnalyticsCollectionCreated(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"Created analytics collection"}, nil, collectionsRegexp, 3)
	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:               values.AnalyticsCollectionCreatedEvent,
		Collection:          output[2],
		AnalyticsCollection: output[1],
	}, nil
}

// AnalyticsCollectionDropped gets when an analytics collection is dropped.
// Example line: 2021-05-20T10:11:46.666+01:00, analytics:0:info:message(n_3@127.0.0.1) - Dropped analytics collection
//
//	Default.bee
func AnalyticsCollectionDropped(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"Dropped analytics collection"}, nil, collectionRegexp, 2)
	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:               values.AnalyticsCollectionDroppedEvent,
		AnalyticsCollection: output[1],
	}, nil
}

// AnalyticsIndexCreatedOrDropped gets when an analytics index is added to or dropped from a collection/dataset.
// Example dataset created line: 2021-02-19T03:24:37.959-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Created
//
//	index "idx_1" on shadow dataset "ds_3".
//
// Example collection created line: 2021-05-13T11:26:25.877+01:00, analytics:0:info:message(n_1@127.0.0.1) - Created
//
//	index brewerIndex on analytics collection Default.test
//
// Example dropped line: 2021-05-20T09:43:36.032+01:00, analytics:0:info:message(n_3@127.0.0.1) - Dropped index
//
//	beers_name_idxfrom analytics collection Default.beers
func AnalyticsIndexCreatedOrDropped(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"analytics"}, []string{"Created index", "Dropped index"},
		indexRegexp, 3)
	if err != nil {
		return nil, err
	}

	event := values.AnalyticsIndexCreatedEvent
	if strings.Contains(line, "Dropped index") {
		event = values.AnalyticsIndexDroppedEvent
	}

	if output[2] == "collection" {
		return &values.Result{
			Event:               event,
			AnalyticsCollection: output[3],
			Index:               output[1],
		}, nil
	}
	return &values.Result{
		Event:   event,
		Dataset: output[3],
		Index:   output[1],
	}, nil
}

// AnalyticsScopeCreated gets when ananalytics scope is created or dropped.
// Example created line: 2021-05-20T10:26:18.008+01:00, analytics:0:info:message(n_3@127.0.0.1) - Created analytics
//
//	scope bee
//
// Example dropped line: 2021-05-20T10:26:27.577+01:00, analytics:0:info:message(n_3@127.0.0.1) - Dropped analytics
//
//	scope bee
func AnalyticsScopeCreatedOrDropped(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, nil, []string{"Created analytics scope", "Dropped analytics scope"},
		scopeRegexp, 2)
	if err != nil {
		return nil, err
	}

	event := values.AnalyticsScopeCreatedEvent
	if strings.Contains(line, "Dropped analytics scope") {
		event = values.AnalyticsScopeDroppedEvent
	}

	return &values.Result{
		Event:          event,
		AnalyticsScope: output[1],
	}, nil
}

// LinkConnectedOrDisconnected gets when an analytics link is connected or disconnected.
// Example connected line: 2021-05-20T09:40:49.143+01:00, analytics:0:info:message(n_3@127.0.0.1) - Connected bucket
//
//	beer-sample for link Default.Local
//
// Example disconnected line: 2021-05-13T11:30:35.579+01:00, analytics:0:info:message(n_1@127.0.0.1) - Disconnected
//
//	link Default.Local
func LinkConnectedOrDisconnected(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"analytics", "link"},
		[]string{"Connected", "Disconnected link"}, linkRegexp, 2)
	if err != nil {
		return nil, err
	}

	event := values.AnalyticsLinkConnectedEvent
	if strings.Contains(line, "Disconnected link") {
		event = values.AnalyticsLinkDisconnectedEvent
	}

	return &values.Result{
		Event: event,
		Link:  output[1],
	}, nil
}

// DataverseCreatedOrDropped gets when a dataverse is created or dropped.
// Example line: 2021-02-19T03:19:26.330-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Created dataverse
//
//	"dv_1".
func DataverseCreatedOrDropped(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"analytics"},
		[]string{"Created dataverse", "Dropped dataverse"}, dataverseRegexp, 2)
	if err != nil {
		return nil, err
	}

	event := values.DataverseCreatedEvent
	if strings.Contains(line, "Dropped dataverse") {
		event = values.DataverseDroppedEvent
	}

	return &values.Result{
		Event:     event,
		Dataverse: output[1],
	}, nil
}

// DatasetCreated gets when an analytics dataset is created.
// Example line: 2021-02-19T03:19:33.248-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Created dataset "ds_1"
//
//	on bucket "bucket6".
func DatasetCreated(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"analytics", "Created dataset"}, nil, datasetCreatedRegexp,
		3)
	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:   values.DatasetCreatedEvent,
		Bucket:  output[2],
		Dataset: output[1],
	}, nil
}

// DatasetDropped gets when an analytics dataset is dropped.
// Example line: 2020-06-22T14:34:21.533Z, analytics:0:info:message(ns_1@10.17.124.92) - Dropped dataset "crew_effects".
func DatasetDropped(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"analytics", "Dropped dataset"}, nil, datasetDroppedRegexp,
		2)
	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:   values.DatasetDroppedEvent,
		Dataset: output[1],
	}, nil
}
