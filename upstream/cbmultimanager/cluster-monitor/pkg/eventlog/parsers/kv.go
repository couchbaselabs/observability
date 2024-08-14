// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package parsers

import (
	"errors"
	"regexp"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"
)

var (
	// capture the bucket name and type when a bucket is created
	createdBucketRegexp = regexp.MustCompile(`Created\sbucket\s"(?P<bucket>[^"]*)".*bucket_type=(?P<bucket_type>[^;]*);`)
	// capture the bucket name when a bucket is deleted
	deletedBucketRegexp = regexp.MustCompile(`Deleted\sbucket\s"(?P<bucket>[^"]*)"`)
	// capture the bucket name and config string when a bucket is updated
	bucketUpdatedRegexp = regexp.MustCompile(`Updated\sbucket\s"(?P<bucket>[^"]*)".*properties:\[(?P<config>[^\]]*)\]`)
	// capture anything between {}
	bracketRegexp = regexp.MustCompile(`\{[^\}]*\}`)
	// capture the bucket and node names when a bucket is flushed
	flushedRegexp = regexp.MustCompile(`Flushing\sbucket\s"(?P<bucket>[^"]*)".*node\s'(?P<node>[^']*)'`)
	// capture whether the collection is being created or dropped along with the scope, collection and bucket
	kvCollectionRegexp = regexp.MustCompile(`(?P<type>create|drop)_collection,"(?P<scope>[^"]*)","(?P<collection>[^"]*)"` +
		`.*bucket\s"(?P<bucket>[^"]*)"`)
	// capture whether the scope is being created or dropped along with the scope and bucket
	kvScopeRegexp = regexp.MustCompile(`(?P<type>create|drop)_scope,"(?P<scope>[^"]*)".*bucket\s"(?P<bucket>[^"]*)"`)
)

// BucketCreated gets when a bucket was created along with all of the config parameters.
func BucketCreated(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"do_ensure_bucket", "Created bucket"}, nil, createdBucketRegexp, 3)
	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:      values.BucketCreatedEvent,
		Bucket:     output[1],
		BucketType: output[2],
	}, nil
}

// BucketDeleted gets when a bucket was deleted.
func BucketDeleted(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"Deleted bucket"}, nil, deletedBucketRegexp, 2)
	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:  values.BucketDeletedEvent,
		Bucket: output[1],
	}, nil
}

// BucketUpdated gets when a bucket was updated.
func BucketUpdated(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"Updated bucket"}, nil, bucketUpdatedRegexp, 3)
	if err != nil {
		return nil, err
	}

	// convert format: [{num_replicas,0},{ram_quota,209715200},{flush_enabled,false},{storage_mode,couchstore}]
	// into format: map[string]string{"num_replicas":  "0", "ram_quota": "209715200", "flush_enabled": "false",
	//		"storage_mode":  "couchstore"}
	config := output[2]
	// get a list of settings
	configList := bracketRegexp.FindAllString(config, -1)
	settings := make(map[string]string)

	for _, setting := range configList {
		// remove brackets
		setting = setting[1 : len(setting)-1]
		// split name from value
		settingSlice := strings.Split(setting, ",")
		if len(settingSlice) < 2 {
			return nil, errors.New("missing config parameter value")
		}

		// add setting to map
		settings[settingSlice[0]] = settingSlice[1]
	}

	return &values.Result{
		Event:    values.BucketUpdatedEvent,
		Bucket:   output[1],
		Settings: settings,
	}, nil
}

// BucketFlushed gets when a bucket was flushed.
func BucketFlushed(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"Flushing bucket"}, nil, flushedRegexp, 3)
	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:  values.BucketFlushedEvent,
		Bucket: output[1],
		Node:   output[2],
	}, nil
}

// AddOrDropScope gets when a a scope is created or dropped.
// Example added line: [ns_server:debug,2021-04-08T12:38:02.366Z,ns_1@10.144.210.101:<0.22560.86>:collections:do_update_
//
//	as_leader:252]Perform operation {create_scope,"scope_test"} on manifest [{uid,7},{next_uid,7},{next_scope_uid,9},
//	{next_coll_uid,10},{num_scopes,1},{num_collections,2},{scopes,[{"test_scope",[{uid,8},{collections,[{"coll-in-scope"
//	,[{uid,10}]}]}]},{"_default",[{uid,0},{collections,[{"test-coll",[{uid,8}]},{"_default",[{uid,0}]}]}]}]}] of bucket
//	"test"
//
// Example dropped line: [ns_server:debug,2021-04-08T12:49:13.357Z,ns_1@10.144.210.101:<0.2841.91>:collections:do_update
//
//	_as_leader:252]Perform operation {drop_scope,"scope_test"} on manifest [{uid,9},{next_uid,9},{next_scope_uid,10},
//	{next_coll_uid,11},{num_scopes,2},{num_collections,3},{scopes,[{"scope_test",[{uid,10},{collections,[]}]},
//	{"test_scope",[{uid,8},{collections,[{"coll_test",[{uid,11}]},{"coll-in-scope",[{uid,10}]}]}]},{"_default",[{uid,0},
//	{collections,[{"test-coll",[{uid,8}]},{"_default",[{uid,0}]}]}]}]}] of bucket "test"
func AddOrDropScope(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, nil,
		[]string{"Perform operation {create_scope,", "Perform operation {drop_scope,"}, kvScopeRegexp, 4)
	if err != nil {
		return nil, err
	}

	event := values.ScopeAddedEvent
	if output[1] == "drop" {
		event = values.ScopeDroppedEvent
	}

	return &values.Result{
		Event:  event,
		Bucket: output[3],
		Scope:  output[2],
	}, nil
}

// AddOrDropCollection gets when a collection is created or dropped.
// Example added line: [ns_server:debug,2021-04-08T12:38:17.615Z,ns_1@10.144.210.101:<0.26424.86>:collections:do_update_
//
//	as_leader:252]Perform operation {create_collection,"test_scope","coll_test",[]} on manifest [{uid,8},{next_uid,8},
//	{next_scope_uid,10},{next_coll_uid,10},{num_scopes,2},{num_collections,2},{scopes,[{"scope_test",[{uid,10},
//	{collections,[]}]},{"test_scope",[{uid,8},{collections,[{"coll-in-scope",[{uid,10}]}]}]},{"_default",[{uid,0},
//	{collections,[{"test-coll",[{uid,8}]},{"_default",[{uid,0}]}]}]}]}] of bucket "test
//
// Example dropped line: [ns_server:debug,2021-04-08T12:49:19.942Z,ns_1@10.144.210.101:<0.5078.91>:collections:do_
//
//	update_as_leader:252]Perform operation {drop_collection,"test_scope","coll_test"} on manifest [{uid,10},{next_uid
//	,10},{next_scope_uid,10},{next_coll_uid,11},{num_scopes,1},{num_collections,3},{scopes,[{"test_scope",[{uid,8},
//	{collections,[{"coll_test",[{uid,11}]},{"coll-in-scope",[{uid,10}]}]}]},{"_default",[{uid,0},{collections,[
//	{"test-coll",[{uid,8}]},{"_default",[{uid,0}]}]}]}]}] of bucket "test"
func AddOrDropCollection(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, nil,
		[]string{"Perform operation {create_collection", "Perform operation {drop_collection,"}, kvCollectionRegexp, 5)
	if err != nil {
		return nil, err
	}

	event := values.CollectionAddedEvent
	if output[1] == "drop" {
		event = values.CollectionDroppedEvent
	}

	return &values.Result{
		Event:      event,
		Bucket:     output[4],
		Scope:      output[2],
		Collection: output[3],
	}, nil
}
