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

// XDCRReplicationCreatedOrRemovedStart gets when an XDCR replication starts to be created or removed.
func XDCRReplicationCreatedOrRemovedStart(line string) (*values.Result, error) {
	if !strings.Contains(line, "xdcr") || !strings.Contains(line, "Replication") || !strings.Contains(line, "user") ||
		!(strings.Contains(line, "created") || strings.Contains(line, "removed")) {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`from\sbucket\s"(?P<source>[^"]*)"\sto\sbucket\s"(?P<target>[^"]*)"\son\scluster` +
		`\s"(?P<cluster>[^"]*)"`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 4 {
		return nil, values.ErrRegexpMissingFields
	}

	event := values.XDCRReplicationCreateStartedEvent
	if strings.Contains(line, "removed") {
		event = values.XDCRReplicationRemoveStartedEvent
	}

	return &values.Result{
		Event:        event,
		SourceBucket: output[0][1],
		TargetBucket: output[0][2],
		Cluster:      output[0][3],
	}, nil
}

// XDCRReplicationCreateOrRemoveFailed gets when an XDCR replication creation or removal fails.
func XDCRReplicationCreateOrRemoveFailed(line string) (*values.Result, error) {
	if !strings.Contains(line, "Error creating replication") && !strings.Contains(line, "Error deleting replication") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`Last\serror:\s(?P<error>[^\]]*)\]`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 2 {
		return nil, values.ErrRegexpMissingFields
	}

	event := values.XDCRReplicationCreateFailedEvent
	if strings.Contains(line, "Error deleting replication") {
		event = values.XDCRReplicationRemoveFailedEvent
	}

	return &values.Result{
		Event:  event,
		Reason: output[0][1],
	}, nil
}

// XDCRReplicationCreateOrRemoveSuccess gets when an XDCR replication creation or removal succeeds.
func XDCRReplicationCreateOrRemoveSuccess(line string) (*values.Result, error) {
	if !strings.Contains(line, "Replication specification") ||
		(!strings.Contains(line, "created") && !strings.Contains(line, "deleted")) {
		return nil, values.ErrNotInLine
	}

	event := values.XDCRReplicationCreateSuccessfulEvent
	if strings.Contains(line, "deleted") {
		event = values.XDCRReplicationRemoveSuccessfulEvent
	}

	return &values.Result{
		Event: event,
	}, nil
}
