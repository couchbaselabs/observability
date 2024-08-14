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

// FTSIndexCreatedOrDropped gets when a full-text index is created or dropped.
func FTSIndexCreatedOrDropped(line string) (*values.Result, error) {
	if !strings.Contains(line, "index definition created") && !strings.Contains(line, "index definition deleted") {
		return nil, values.ErrNotInLine
	}

	lineRegexp := regexp.MustCompile(`indexName:\s(?P<index>[^,]*),`)
	output := lineRegexp.FindAllStringSubmatch(line, 1)
	if len(output) == 0 || len(output[0]) < 2 {
		return nil, values.ErrRegexpMissingFields
	}

	event := values.FTSIndexDroppedEvent
	if strings.Contains(line, "index definition created") {
		event = values.FTSIndexCreatedEvent
	}

	return &values.Result{
		Event: event,
		Index: output[0][1],
	}, nil
}
