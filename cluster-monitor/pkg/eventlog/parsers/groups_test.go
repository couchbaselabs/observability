// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package parsers

import (
	"regexp"
	"testing"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"

	"github.com/stretchr/testify/require"
)

type captureGroupTestCase struct {
	Name           string
	Line           string
	ExpectedResult []string
	ExpectedError  error
	Regexp         *regexp.Regexp
	IncludeGroups  []string
	OneOfGroup     []string
	FieldNum       int
}

func TestGetCaptureGroups(t *testing.T) {
	testCases := []captureGroupTestCase{
		{
			Name: "inLine",
			Line: `2020-06-22T14:34:21.533Z, analytics:0:info:message(ns_1@10.17.124.92) - Dropped dataset ` +
				`"crew_effects".`,
			IncludeGroups:  []string{"analytics", "Dropped dataset"},
			Regexp:         regexp.MustCompile(`dataset\s"(?P<datatset>[^"]*)"`),
			ExpectedResult: []string{"dataset \"crew_effects\"", "crew_effects"},
			FieldNum:       2,
		},
		{
			Name: "multiFieldLine",
			Line: `2021-02-19T03:19:33.248-08:00, analytics:0:info:message(ns_1@172.23.106.188) - Created dataset "ds_1"` +
				` on bucket "bucket6".`,
			IncludeGroups:  []string{"analytics", "Created dataset"},
			Regexp:         regexp.MustCompile(`dataset\s"(?P<datatset>[^"]*)".*bucket\s"(?P<bucket>[^"]*)"`),
			ExpectedResult: []string{`dataset "ds_1" on bucket "bucket6"`, "ds_1", "bucket6"},
			FieldNum:       2,
		},
		{
			Name: "oneOfFieldLine",
			Line: `2021-05-20T10:26:18.008+01:00, analytics:0:info:message(n_3@127.0.0.1) - Created analytics ` +
				`scope bee`,
			OneOfGroup:     []string{"Created analytics scope", "Dropped analytics scope"},
			Regexp:         regexp.MustCompile(`analytics\sscope\s(?P<scopeA>.*)`),
			ExpectedResult: []string{`analytics scope bee`, "bee"},
			FieldNum:       2,
		},
		{
			Name: "notInLine",
			Line: `2020-06-22T14:34:21.533Z, analytics:0:info:message(ns_1@10.17.124.92) - Dropped data ` +
				`"crew_effects".`,
			IncludeGroups: []string{"analytics", "Dropped dataset"},
			Regexp:        regexp.MustCompile(`dataset\s"(?P<datatset>[^"]*)"`),
			ExpectedError: values.ErrNotInLine,
			FieldNum:      2,
		},
		{
			Name: "oneOfFieldLineNotInLine",
			Line: `2021-05-20T10:26:18.008+01:00, analytics:0:info:message(n_3@127.0.0.1) - Created analytics ` +
				`scope bee`,
			OneOfGroup:    []string{"Created scope"},
			Regexp:        regexp.MustCompile(`analytics\sscope\s(?P<scopeA>.*)`),
			ExpectedError: values.ErrNotInLine,
			FieldNum:      2,
		},
	}

	for _, x := range testCases {
		t.Run(x.Name, func(t *testing.T) {
			inLineResult, err := getCaptureGroups(x.Line, x.IncludeGroups, x.OneOfGroup, x.Regexp, x.FieldNum)
			require.Equal(t, x.ExpectedResult, inLineResult)
			require.ErrorIs(t, err, x.ExpectedError)
		})
	}
}
