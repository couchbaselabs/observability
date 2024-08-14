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
	// createIndexRegexp7_0_0 captures the settings of an index when it is created using scopes and collections
	createIndexRegexp7_0_0 = regexp.MustCompile(`Create\sIndex\sDefnId:\s(?P<DefnId>[^\s]*)\sName:\s(?P<Name>[^\s]*)\s` +
		`Using:\s(?P<Using>[^\s]*)\sBucket:\s(?P<Bucket>[^\s]*)\sScope\/Id:\s(?P<ScopeId>[^\s]*)\sCollection\/Id:\s` +
		`(?P<CollectionId>[^\s]*)\sIsPrimary:\s(?P<IsPrimary>[^\s]*)\sNumReplica:\s(?P<NumReplica>[^\s]*)\sInstVersion:` +
		`\s(?P<InstVersion>.*)`)
	// createIndexRegexp5_0_0To6_6_0 captures the settings of an index when it is created without using scopes and
	//	collections
	createIndexRegexp5_0_0To6_6_0 = regexp.MustCompile(`Create\sIndex\sDefnId:\s(?P<DefnId>[^\s]*)\sName:\s(?P<Name>[` +
		`^\s]*)\sUsing:\s(?P<Using>[^\s]*)\sBucket:\s(?P<Bucket>[^\s]*)\sIsPrimary:\s(?P<IsPrimary>[^\s]*)\sNumReplica:\s` +
		`(?P<NumReplica>[^\s]*)\sInstVersion:\s(?P<InstVersion>.*)`)
	// deleteIndexRegexp captures the name of the index
	deleteIndexRegexp = regexp.MustCompile(`IndexId\s(?P<index>.*)`)
)

// IndexCreated gets when a query index was created.
func IndexCreated(line string) (*values.Result, error) {
	lineRegexp := createIndexRegexp5_0_0To6_6_0
	if strings.Contains(line, "Scope/Id") {
		lineRegexp = createIndexRegexp7_0_0
	}

	output, err := getCaptureGroups(line, []string{"OnIndexCreate", "Success"}, nil, lineRegexp,
		len(lineRegexp.SubexpNames()))
	if err != nil {
		return nil, err
	}

	jsonSettings := make(map[string]string)
	for i, name := range lineRegexp.SubexpNames()[1:] {
		jsonSettings[name] = output[i+1]
	}

	return &values.Result{
		Event:    values.IndexCreatedEvent,
		Settings: jsonSettings,
	}, nil
}

// IndexDeleted gets when a query index was deleted.
func IndexDeleted(line string) (*values.Result, error) {
	if !strings.Contains(line, ":OnIndexDelete") || strings.Contains(line, "Notification Received") {
		return nil, values.ErrNotInLine
	}

	output := deleteIndexRegexp.FindStringSubmatch(line)
	if len(output) < 2 {
		return nil, values.ErrRegexpMissingFields
	}

	return &values.Result{
		Event: values.IndexDeletedEvent,
		Index: output[1],
	}, nil
}

// IndexerActive gets when the indexer becomes active.
func IndexerActive(line string) (*values.Result, error) {
	if !strings.Contains(line, "NewIndexer Status Active") {
		return nil, values.ErrNotInLine
	}

	return &values.Result{
		Event: values.IndexerActiveEvent,
	}, nil
}
