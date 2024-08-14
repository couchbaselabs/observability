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

func getCaptureGroups(line string, necesseryFields []string, oneOfFields []string, regex *regexp.Regexp,
	mustCapture int,
) ([]string, error) {
	for _, field := range necesseryFields {
		if !strings.Contains(line, field) {
			return nil, values.ErrNotInLine
		}
	}

	contains := len(oneOfFields) == 0

	for _, field := range oneOfFields {
		if strings.Contains(line, field) {
			contains = true
			break
		}
	}

	if !contains {
		return nil, values.ErrNotInLine
	}

	output := regex.FindStringSubmatch(line)
	if len(output) < mustCapture {
		return nil, values.ErrRegexpMissingFields
	}

	return output, nil
}
